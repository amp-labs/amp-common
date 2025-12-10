package cli

import (
	"strings"

	"github.com/amp-labs/amp-common/hashing"
	"github.com/amp-labs/amp-common/set"
	"github.com/manifoldco/promptui"
)

// MultiSelect displays an interactive multi-selection menu where users can choose multiple items.
// The user can search/filter choices by typing, select items one at a time, and choose "[Done]" when finished.
// Returns the selected items in their original order from the choices slice.
// Returns nil if no choices are provided.
func MultiSelect(label string, choices ...string) ([]string, error) {
	if len(choices) == 0 {
		return nil, nil
	}

	allNames := set.NewStringSet(hashing.Sha256)

	if err := allNames.AddAll(choices...); err != nil { //nolint:noinlineerr // Inline error handling is clear here
		return nil, err
	}

	names := allNames.SortedEntries()

	selections := set.NewStringSet(hashing.Sha256)

	names = append([]string{"[Done]"}, names...)

again:
	sel := &promptui.Select{
		Label: label,
		Items: names,
		Searcher: func(input string, index int) bool {
			if index == 0 {
				return false
			}

			if len(input) == 0 {
				return false
			}

			n := names[index]

			return strings.HasPrefix(n, input)
		},
	}

	idx, value, err := sel.Run()
	if err != nil {
		return nil, err
	}

	if idx != 0 {
		err := selections.Add(value)
		if err != nil {
			return nil, err
		}

		err = allNames.Remove(value)
		if err != nil {
			return nil, err
		}

		names = allNames.SortedEntries()
		if len(names) > 0 {
			names = append([]string{"[Done]"}, names...)

			goto again
		}
	}

	var choicesOut []string

	for _, c := range choices {
		contains, err := selections.Contains(c)
		if err != nil {
			return nil, err
		}

		if contains {
			choicesOut = append(choicesOut, c)
		}
	}

	return choicesOut, nil
}
