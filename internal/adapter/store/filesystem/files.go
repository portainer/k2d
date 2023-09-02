package filesystem

import (
	"os"
	"strings"
)

// containsFileWithPrefix checks if there is at least one file in the given list that has a filename starting with the specified prefix.
//
// Parameters:
//   - files: A slice of os.DirEntry objects representing the list of files to be checked.
//   - filePrefix: The prefix to be used for matching the filenames.
//
// Returns:
//   - bool: Returns true if at least one file with the specified prefix is found; otherwise, returns false.
func containsFileWithPrefix(files []os.DirEntry, filePrefix string) bool {
	for _, file := range files {
		if strings.HasPrefix(file.Name(), filePrefix) {
			return true
		}
	}
	return false
}
