package utils

import (
	"log/slog"
	"strings"
)

func GetOrdinalNumberInt(s string) uint8 {
	switch strings.ToLower(s) {
	case "prima", "primo":
		return 1

	case "secondo", "seconda":
		return 2

	case "terzo", "terza":
		return 3

	case "quarto", "quarta":
		return 4

	case "quinto", "quinta":
		return 5

	case "sesto", "sesta":
		return 6

	case "settimo", "settima":
		return 7

	case "ottavo", "ottava":
		return 8

	case "nono", "nona":
		return 9

	case "decimo", "decima":
		return 10

	default:
		slog.Error("[GetOrdinalNumberInt] unrecognized number string", "string", s)
		return 0
	}
}
