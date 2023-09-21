package prompt

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// AskForConfirmation asks the user for a confirmation.
func AskForConfirmation() (bool, error) {
	r := bufio.NewReader(os.Stdin)
	line, _, err := r.ReadLine()
	if err != nil {
		return false, err
	}

	response := string(line)

	switch strings.ToLower(response) {
	case "y", "yes":
		return true, nil
	case "n", "no":
		return false, nil
	case "":
		return false, nil
	default:
		fmt.Println("Please type [y]es or [n]o and then press enter:")
		return AskForConfirmation()
	}
}
