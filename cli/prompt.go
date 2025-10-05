package cli

import (
	"errors"
	"fmt"
	"os"
	"strconv"

	"github.com/manifoldco/promptui"
)

var errEmptyInput = errors.New("you must enter something")

// PromptConfirm displays a yes/no confirmation prompt to the user.
// Returns true if the user confirms (presses 'y' or Enter), false if they decline ('n').
// If the user aborts (Ctrl+C), returns (false, nil).
func PromptConfirm(label string) (bool, error) {
	prompt := promptui.Prompt{
		Label:     label,
		IsConfirm: true,
		Stdin:     os.Stdin,
		Stdout:    os.Stdout,
	}

	_, err := prompt.Run()
	if err != nil {
		if errors.Is(err, promptui.ErrAbort) {
			return false, nil
		}

		return false, err
	}

	return true, nil
}

// PromptString displays a text input prompt and requires non-empty input.
// Returns an error if the user provides empty input.
func PromptString(label string) (string, error) {
	prompt := promptui.Prompt{
		Label: label,
		Validate: func(s string) error {
			if len(s) == 0 {
				return errEmptyInput
			}

			return nil
		},
		Stdin:  os.Stdin,
		Stdout: os.Stdout,
	}

	return prompt.Run()
}

// PromptInt displays a text input prompt and validates that the input is a valid integer.
// Returns the parsed integer value or an error if parsing fails.
func PromptInt(label string) (int, error) {
	prompt := promptui.Prompt{
		Label: label,
		Validate: func(s string) error {
			_, err := strconv.ParseInt(s, 10, 32)
			if err != nil {
				return fmt.Errorf("invalid integer: %w", err)
			}

			return nil
		},
		Stdin:  os.Stdin,
		Stdout: os.Stdout,
	}

	txt, err := prompt.Run()
	if err != nil {
		return 0, err
	}

	val, err := strconv.ParseInt(txt, 10, 32)
	if err != nil {
		return 0, fmt.Errorf("invalid integer: %w", err)
	}

	return int(val), nil
}

// PromptStringEmptyOk displays a text input prompt that allows empty input.
// Unlike PromptString, this function accepts empty strings as valid input.
func PromptStringEmptyOk(label string) (string, error) {
	prompt := promptui.Prompt{
		Label:  label,
		Stdin:  os.Stdin,
		Stdout: os.Stdout,
	}

	return prompt.Run()
}
