package main

import (
	"log/slog"

	"github.com/PoliNetworkOrg/rankings-backend-go/pkg/logger"
)

func main() {
	slog.SetDefault(logger.GetDefaultLogger())

	slog.Warn("THIS IS A DEV-ONLY PURPOSE MODULE, DON'T USE IN PRODUCTION")

	// t := time.NewTicker(500 * time.Millisecond)
	// for {
	// 	select {
	// 		case <- t.C:
	// 		slog.Debug("tick", "time", t)
	// 	}
	// }

	JsonEncodeDecodeStruct()
}
