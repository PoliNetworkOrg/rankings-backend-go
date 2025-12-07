package parser

import (
	"cmp"
	"slices"
)

// langPriority converts your language string into a comparable integer.
// "IT" (1) < "EN" (2) < Others (3)
func langPriority(lang string) int {
	switch lang {
	case "IT":
		return 1
	case "EN":
		return 2
	default:
		return 3
	}
}

// boolToInt is a helper for sorting bools since cmp.Compare doesn't support them natively yet.
// false = 0, true = 1
func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func SortPhases(phases []Phase) {
	slices.SortStableFunc(phases, CmpPhases)
}

func CmpPhases(a, b Phase) int {
	return cmp.Or(
		cmp.Compare(a.Primary, b.Primary),
		cmp.Compare(a.Secondary, b.Secondary),
		cmp.Compare(langPriority(a.Language), langPriority(b.Language)),
		cmp.Compare(boolToInt(a.IsExtraEu), boolToInt(b.IsExtraEu)),
	)
}
