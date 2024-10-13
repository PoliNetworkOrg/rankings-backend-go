package main

import (
	"log/slog"
	"github.com/PoliNetworkOrg/rankings-backend-go/pkg/logger"
)

func main() {
	slog.SetDefault(logger.GetDefaultLogger())
	opts := ParseOpts()

	slog.Info("argv validation", "data_dir", opts.dataDir)
}
