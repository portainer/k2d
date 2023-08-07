package filesystem

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// FileExists checks if a file exists on the filesystem
func FileExists(path string) (bool, error) {
	if _, err := os.Stat(path); err == nil {
		return true, nil
	} else if errors.Is(err, os.ErrNotExist) {
		return false, nil
	} else {
		return false, fmt.Errorf("an error occurred while checking if the file exists: %w", err)
	}
}

// CreateDir creates all directories along a path.
// It returns an error, if any occurs during the operation.
func CreateDir(path string) error {
	err := os.MkdirAll(path, 0755)
	if err != nil {
		return err
	}
	return nil
}

// CreateFileWithDirectories creates a new file at the specified filePath
// and writes the provided content to it. If the file's directory path doesn't
// exist, it will be created automatically.
//
// Parameters:
// - filePath: The path where the file should be created, including the file's name.
// - content: The content that should be written to the file as a byte array.
//
// It returns an error if any filesystem operation fails.
func CreateFileWithDirectories(filePath string, content []byte) error {
	// Ensure the parent directory exists
	dirPath := filepath.Dir(filePath)
	if err := os.MkdirAll(dirPath, 0755); err != nil {
		return fmt.Errorf("unable to create directory path: %w", err)
	}

	// Create and write to the file
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("unable to create file: %w", err)
	}
	defer file.Close()

	_, err = file.Write(content)
	if err != nil {
		return fmt.Errorf("unable to write to file: %w", err)
	}

	return nil
}

// StoreDataMapOnDisk takes a path where the data will be stored (storagePath), a prefix for the filename (filePrefix),
// and a map of strings (data). It iterates through the provided map, and for each key-value pair,
// it creates a file with the filename constructed as the concatenation of the filePrefix and the key.
// It then writes the corresponding value into the file.
func StoreDataMapOnDisk(storagePath, filePrefix string, data map[string]string) error {
	for key, value := range data {

		fileName := fmt.Sprintf("%s%s", filePrefix, key)

		file, err := os.Create(path.Join(storagePath, fileName))
		if err != nil {
			return fmt.Errorf("an error occurred while creating the file: %w", err)
		}
		defer file.Close()

		_, err = file.WriteString(value)
		if err != nil {
			return fmt.Errorf("an error occurred while writing to the file: %w", err)
		}
	}

	return nil
}

func StoreMetadataOnDisk(storagePath, fileName string, data map[string]string) error {
	file, err := os.Create(path.Join(storagePath, fileName))
	if err != nil {
		return fmt.Errorf("an error occurred while creating the file: %w", err)
	}
	defer file.Close()

	for key, value := range data {
		_, err = file.WriteString(fmt.Sprintf("%s=%s\n", key, value))
		if err != nil {
			return fmt.Errorf("an error occurred while writing to the file: %w", err)
		}
	}

	return nil
}

func LoadMetadataFromDisk(storagePath, fileName string) (map[string]string, error) {
	file, err := os.Open(path.Join(storagePath, fileName))
	if err != nil {
		return nil, fmt.Errorf("an error occurred while opening the file: %w", err)
	}
	defer file.Close()

	data := make(map[string]string)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid data format: %s", line)
		}
		data[parts[0]] = parts[1]
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("an error occurred while reading the file: %w", err)
	}

	return data, nil
}
