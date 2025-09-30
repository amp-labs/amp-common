package cli

import (
	"strings"

	"github.com/amp-labs/amp-common/hashing"
	"github.com/amp-labs/amp-common/set"
	"github.com/manifoldco/promptui"
)

func MultiSelect(label string, choices ...string) ([]string, error) {
	if len(choices) == 0 {
		return nil, nil
	}

	allNames := set.NewStringSet(hashing.Sha256)

	if err := allNames.AddAll(choices...); err != nil {
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
		if err := selections.Add(value); err != nil {
			return nil, err
		}

		if err := allNames.Remove(value); err != nil {
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
