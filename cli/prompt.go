package cli

import (
	"errors"
	"fmt"
	"os"
	"strconv"

	"github.com/manifoldco/promptui"
)

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

func PromptString(label string) (string, error) {
	prompt := promptui.Prompt{
		Label: label,
		Validate: func(s string) error {
			if len(s) == 0 {
				return errors.New("you must enter something")
			}

			return nil
		},
		Stdin:  os.Stdin,
		Stdout: os.Stdout,
	}

	return prompt.Run()
}

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

func PromptStringEmptyOk(label string) (string, error) {
	prompt := promptui.Prompt{
		Label:  label,
		Stdin:  os.Stdin,
		Stdout: os.Stdout,
	}

	return prompt.Run()
}
