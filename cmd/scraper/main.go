package main

import (
	"encoding/json"
	"log/slog"
	"os"

	"github.com/PoliNetworkOrg/rankings-backend-go/pkg/logger"
	"github.com/PoliNetworkOrg/rankings-backend-go/pkg/scraper"
	"github.com/PoliNetworkOrg/rankings-backend-go/pkg/utils"
)

func main() {
	slog.SetDefault(logger.GetDefaultLogger())
	opts := ParseOpts()

	slog.Info("argv validation", "data_dir", opts.dataDir)

	mans := scraper.ScrapeManifesti()
	json, err := json.MarshalIndent(mans, "", "	")
	if err != nil {
		panic(err)
	}

	tmpExists, err := utils.DoFolderExists("tmp")
	if !tmpExists || err != nil {
		os.Mkdir("tmp", os.ModePerm)
	}
	os.WriteFile("tmp/test.json", json, 0644)
}
