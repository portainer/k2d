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
//
// Parameters:
// - volumeName: The target Docker volume where the data will be copied.
// - dataMap: A map where the keys are file names and the values are file contents.
//
// Returns:
// - Returns an error if any step in the pipeline (container creation, data copying, or container removal) fails.
func (s *VolumeStore) copyDataMapToVolume(volumeName string, dataMap map[string]string) error {
	containerConfig := &container.Config{
		Image: s.copyImageName,
	}
	hostConfig := &container.HostConfig{
		Binds: []string{
			fmt.Sprintf("%s:%s", volumeName, WorkingDirName),
		},
	}

	copyContainerName := fmt.Sprintf("k2d-volume-copy-%s-%d", volumeName, time.Now().UnixNano())
	resp, err := s.cli.ContainerCreate(context.TODO(), containerConfig, hostConfig, nil, nil, copyContainerName)
	if err != nil {
		return fmt.Errorf("unable to create temporary volume copy container: %w", err)
	}

	err = s.cli.ContainerStart(context.TODO(), resp.ID, types.ContainerStartOptions{})
	if err != nil {
		return fmt.Errorf("unable to start temporary volume copy container: %w", err)
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

	err = s.cli.CopyToContainer(context.TODO(), resp.ID, WorkingDirName, &buf, types.CopyToContainerOptions{})
	if err != nil {
		return fmt.Errorf("unable to copy data to temporary volume copy container: %w", err)
	}

	err = s.cli.ContainerRemove(context.Background(), resp.ID, types.ContainerRemoveOptions{
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
// - volumeBindings: A list of volume bindings, which are strings that specify the volumes to attach to the container.
// - containerName: The name to give to the temporary container.
//
// Returns:
// - The ID of the newly created container or an error if the container creation fails.
func (s *VolumeStore) createAndStartCopyContainer(volumeBindings []string, containerName string) (string, error) {
	containerConfig := &container.Config{
		Image: s.copyImageName,
	}
	hostConfig := &container.HostConfig{
		Binds: volumeBindings,
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

// getDataMapFromVolume extracts the data stored in a specific Docker volume and returns it as a map.
// This function creates a temporary container with the volume mounted to extract the data.
//
// Parameters:
// - volumeName: The name of the Docker volume from which to extract data.
//
// Returns:
// - A map where the keys are filenames and the values are file contents, or an error if the operation fails.
func (store *VolumeStore) getDataMapFromVolume(volumeName string) (map[string]string, error) {
	copyContainerName := fmt.Sprintf("k2d-volume-read-%s-%d", volumeName, time.Now().UnixNano())
	containerID, err := store.createAndStartCopyContainer([]string{fmt.Sprintf("%s:%s", volumeName, WorkingDirName)}, copyContainerName)
	if err != nil {
		return nil, err
	}

	content, _, err := store.cli.CopyFromContainer(context.TODO(), containerID, WorkingDirName)
	if err != nil {
		return nil, err
	}

	err = store.cli.ContainerRemove(context.Background(), containerID, types.ContainerRemoveOptions{Force: true})
	if err != nil {
		return nil, err
	}

	return parseTarToMap(content, store.encryptionKey)
}

// getDataMapsFromVolumes extracts the data stored in multiple Docker volumes and returns it as a map of maps.
// It creates a single temporary container, mounts all specified volumes, and then extracts data from them.
//
// Parameters:
// - volumeNames: A list of Docker volume names from which to extract data.
//
// Returns:
// - A map where each key is a volume name and the corresponding value is a map containing that volume's data.
// - An error if the operation fails.
func (store *VolumeStore) getDataMapsFromVolumes(volumeNames []string) (map[string]map[string]string, error) {
	var binds []string
	for _, volumeName := range volumeNames {
		binds = append(binds, fmt.Sprintf("%s:%s", volumeName, path.Join(WorkingDirName, volumeName)))
	}

	copyContainerName := fmt.Sprintf("k2d-volume-read-%d", time.Now().UnixNano())
	containerID, err := store.createAndStartCopyContainer(binds, copyContainerName)
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
// the corresponding value is the file's content. This function iterates through each entry in the TAR archive,
// extracts the file contents, and populates the map.
//
// Parameters:
// - content: An io.Reader representing the TAR content.
//
// Returns:
// - A map representing the extracted files and their contents, or an error if the operation fails.
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
