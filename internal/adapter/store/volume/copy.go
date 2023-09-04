package volume

import (
	"archive/tar"
	"bytes"
	"context"
	"fmt"
	"io"
	"path"
	"path/filepath"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/portainer/k2d/pkg/crypto"
)

// copyDataMapToVolume is responsible for copying a given data map into a specified Docker volume.
// It creates a temporary container, mounts the volume, and then populates it with data.
// If an encryption key is provided, it also encrypts the data before copying.
//
// Parameters:
// - volumeName: The target Docker volume where the data will be copied.
// - dataMap: A map where the keys are file names and the values are file contents.
//
// Returns:
// - Returns an error if any step in the pipeline (container creation, data encryption, data copying, or container removal) fails.
//
// Implementation Details:
// - Creates a temporary container for data copying, with the target Docker volume mounted.
// - Optionally encrypts the data using the encryption key, if provided.
// - Writes the (possibly encrypted) data to a tar archive.
// - Copies the tar archive to the temporary container.
// - Removes the temporary container after data copying is complete.
func (s *VolumeStore) copyDataMapToVolume(volumeName string, dataMap map[string]string) error {
	volumeBinds := []string{fmt.Sprintf("%s:%s", volumeName, WorkingDirName)}
	copyContainerName := fmt.Sprintf("k2d-volume-copy-%s-%d", volumeName, time.Now().UnixNano())
	containerID, err := s.createAndStartCopyContainer(volumeBinds, copyContainerName)
	if err != nil {
		return fmt.Errorf("unable to create temporary volume copy container: %w", err)
	}

	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)

	for key, value := range dataMap {
		data, err := encryptIfKeyProvided([]byte(value), s.encryptionKey)
		if err != nil {
			return fmt.Errorf("unable to write data: %w", err)
		}

		hdr := &tar.Header{
			Name: key,
			Mode: 0400,
			Size: int64(len(data)),
		}

		if err := tw.WriteHeader(hdr); err != nil {
			return fmt.Errorf("unable to write tar header: %w", err)
		}

		if _, err := tw.Write(data); err != nil {
			return fmt.Errorf("unable to write tar body: %w", err)
		}
	}

	if err := tw.Close(); err != nil {
		return fmt.Errorf("unable to close tar writer: %w", err)
	}

	err = s.cli.CopyToContainer(context.TODO(), containerID, WorkingDirName, &buf, types.CopyToContainerOptions{})
	if err != nil {
		return fmt.Errorf("unable to copy data to temporary volume copy container: %w", err)
	}

	err = s.cli.ContainerRemove(context.TODO(), containerID, types.ContainerRemoveOptions{
		Force: true,
	})
	if err != nil {
		return fmt.Errorf("unable to remove temporary volume copy container: %w", err)
	}

	return nil
}

// createAndStartCopyContainer creates a new Docker container with specified volume bindings.
// The container is used for data copying operations.
//
// Parameters:
// - volumeBinds: A list of volume bindings, which are strings that specify the volumes to attach to the container.
// - containerName: The name to give to the temporary container.
//
// Returns:
// - The ID of the newly created container or an error if the container creation fails.
func (s *VolumeStore) createAndStartCopyContainer(volumeBinds []string, containerName string) (string, error) {
	containerConfig := &container.Config{
		Image: s.copyImageName,
	}
	hostConfig := &container.HostConfig{
		Binds: volumeBinds,
	}

	resp, err := s.cli.ContainerCreate(context.TODO(), containerConfig, hostConfig, nil, nil, containerName)
	if err != nil {
		return "", err
	}

	if err = s.cli.ContainerStart(context.TODO(), resp.ID, types.ContainerStartOptions{}); err != nil {
		return "", err
	}

	return resp.ID, nil
}

// getDataMapFromVolume extracts and optionally decrypts the data stored in a specific Docker volume and returns it as a map.
// This function creates a temporary container with the volume mounted to extract the data.
//
// Parameters:
// - volumeName: The name of the Docker volume from which to extract data.
//
// Returns:
// - A map where the keys are filenames and the values are file contents.
// - An error if the operation fails.
//
// Implementation Details:
// - A temporary container is created to read from the mounted volume.
// - If an encryption key is provided, the data is decrypted before being returned.
func (store *VolumeStore) getDataMapFromVolume(volumeName string) (map[string]string, error) {
	copyContainerName := fmt.Sprintf("k2d-volume-read-%s-%d", volumeName, time.Now().UnixNano())
	volumeBinds := []string{fmt.Sprintf("%s:%s", volumeName, WorkingDirName)}
	containerID, err := store.createAndStartCopyContainer(volumeBinds, copyContainerName)
	if err != nil {
		return nil, err
	}

	content, _, err := store.cli.CopyFromContainer(context.TODO(), containerID, WorkingDirName)
	if err != nil {
		return nil, err
	}

	err = store.cli.ContainerRemove(context.TODO(), containerID, types.ContainerRemoveOptions{Force: true})
	if err != nil {
		return nil, err
	}

	return parseTarToMap(content, store.encryptionKey)
}

// getDataMapsFromVolumes extracts and optionally decrypts the data stored in multiple Docker volumes and returns it as a map of maps.
// A single temporary container is created, multiple volumes are mounted, and data is extracted from them.
//
// Parameters:
// - volumeNames: A list of Docker volume names from which to extract data.
//
// Returns:
// - A map where each key is a volume name and the corresponding value is a map containing that volume's data.
// - An error if the operation fails.
//
// Implementation Details:
// - A single temporary container is created to read from multiple mounted volumes.
// - If an encryption key is provided, the data from each volume is decrypted before being returned.
func (store *VolumeStore) getDataMapsFromVolumes(volumeNames []string) (map[string]map[string]string, error) {
	var volumeBinds []string
	for _, volumeName := range volumeNames {
		volumeBinds = append(volumeBinds, fmt.Sprintf("%s:%s", volumeName, path.Join(WorkingDirName, volumeName)))
	}

	copyContainerName := fmt.Sprintf("k2d-volume-read-%d", time.Now().UnixNano())
	containerID, err := store.createAndStartCopyContainer(volumeBinds, copyContainerName)
	if err != nil {
		return nil, err
	}

	result := make(map[string]map[string]string)
	for _, volumeName := range volumeNames {
		content, _, err := store.cli.CopyFromContainer(context.TODO(), containerID, path.Join(WorkingDirName, volumeName))
		if err != nil {
			return nil, err
		}

		dataMap, err := parseTarToMap(content, store.encryptionKey)
		if err != nil {
			return nil, err
		}

		result[volumeName] = dataMap
	}

	err = store.cli.ContainerRemove(context.Background(), containerID, types.ContainerRemoveOptions{Force: true})
	if err != nil {
		return nil, err
	}

	return result, nil
}

// parseTarToMap takes a TAR archive Reader and converts it into a map where each key is a file name and
// the corresponding value is the file's content. Optionally decrypts the content if an encryption key is provided.
//
// Parameters:
// - content: An io.Reader representing the TAR content.
// - encryptionKey: An optional byte slice used for decrypting the content.
//
// Returns:
// - A map representing the extracted and possibly decrypted files and their contents.
// - An error if the operation fails.
//
// Implementation Details:
// - Iterates through each entry in the TAR archive and extracts the file contents.
// - If an encryption key is provided, decrypts the file contents before adding to the map.
func parseTarToMap(content io.Reader, encryptionKey []byte) (map[string]string, error) {
	dataMap := make(map[string]string)
	tr := tar.NewReader(content)

	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		if hdr.Typeflag == tar.TypeReg {
			buf := new(bytes.Buffer)
			if _, err := io.Copy(buf, tr); err != nil {
				return nil, err
			}

			key := filepath.Base(hdr.Name)
			if key != "" {
				data, err := decryptIfKeyProvided(buf.Bytes(), encryptionKey)
				if err != nil {
					return nil, fmt.Errorf("unable to read data: %w", err)
				}

				dataMap[key] = string(data)
			}
		}
	}

	return dataMap, nil
}

// encryptIfKeyProvided encrypts the given data using the encryptionKey if provided.
func encryptIfKeyProvided(data, encryptionKey []byte) ([]byte, error) {
	if len(encryptionKey) == 0 {
		return data, nil
	}

	encryptedData, err := crypto.Encrypt(data, encryptionKey)
	if err != nil {
		return nil, fmt.Errorf("unable to encrypt data: %w", err)
	}

	return encryptedData, nil
}

// decryptIfKeyProvided decrypts the given data using the encryptionKey if provided.
func decryptIfKeyProvided(data, encryptionKey []byte) ([]byte, error) {
	if len(encryptionKey) == 0 {
		return data, nil
	}

	decryptedData, err := crypto.Decrypt(data, encryptionKey)
	if err != nil {
		return nil, fmt.Errorf("unable to decrypt data: %w", err)
	}
	return decryptedData, nil
}
