package scraper

import "strings"

func isRankingsNews(str string) bool {
	newsTesters := []string{
		"graduatorie", "graduatoria", "punteggi", "tol",
		"immatricolazioni", "immatricolazione", "punteggio",
		"matricola", "nuovi studenti",
	}

	for _, tester := range newsTesters {
		if strings.Contains(str, tester) {
			return true
		}
	}

	return false
}
