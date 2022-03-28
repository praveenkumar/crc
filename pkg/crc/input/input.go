package input

import (
	"fmt"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	crcos "github.com/code-ready/crc/pkg/os"
)

func PromptUserForYesOrNo(message string, force bool) bool {
	if force {
		return true
	}
	if !crcos.RunningInTerminal() {
		return false
	}
	var response string
	fmt.Printf(message + "? [y/N]: ")
	fmt.Scanf("%s", &response)

	return strings.ToLower(response) == "y"
}

// PromptUserForSecret can be used for any kind of secret like image pull
// secret or for password.
func PromptUserForSecret(message string, help string) (string, error) {
	var secret string
	prompt := &survey.Password{
		Message: message,
		Help:    help,
	}
	if err := survey.AskOne(prompt, &secret, nil); err != nil {
		return "", err
	}
	return secret, nil
}
