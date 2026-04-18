package model

import "sort"

// SortedChoicesByIndex converts an indexed choice map into a slice ordered by choice index.
func SortedChoicesByIndex(choiceMap map[int]*Choice) []Choice {
	if len(choiceMap) == 0 {
		return nil
	}

	indexes := make([]int, 0, len(choiceMap))
	for idx := range choiceMap {
		indexes = append(indexes, idx)
	}
	sort.Ints(indexes)

	choices := make([]Choice, 0, len(indexes))
	for _, idx := range indexes {
		choice, ok := choiceMap[idx]
		if !ok || choice == nil {
			continue
		}
		choices = append(choices, *choice)
	}

	return choices
}
