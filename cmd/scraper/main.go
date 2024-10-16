package main

import (
	"encoding/json"
	"log/slog"
	"os"

	"github.com/PoliNetworkOrg/rankings-backend-go/pkg/logger"
	"github.com/PoliNetworkOrg/rankings-backend-go/pkg/scraper"
	"github.com/PoliNetworkOrg/rankings-backend-go/pkg/utils"
	"github.com/PoliNetworkOrg/rankings-backend-go/pkg/writer"
)

func main() {
	slog.SetDefault(logger.GetDefaultLogger())
	opts := ParseOpts()

	slog.Info("argv validation", "data_dir", opts.dataDir)

	mansB, err := os.ReadFile("tmp/test.json")
	var mans []scraper.Manifesto

	// the following is crazy, but atm it's for testing 
	if err != nil || len(mansB) == 0 {
		mans = scraper.ScrapeManifesti()
	} else {
		err = json.Unmarshal(mansB, &mans)
		if err != nil {
			mans = scraper.ScrapeManifesti()
		}
	}

	writer.WriteManifesti(mans)

	equals, err := utils.TestJsonEquals("tmp/test_map.json", opts.dataDir + "/output/manifesti.json")
	if err != nil {
		panic(err)
	}
	slog.Info("scrape manifesti, equals to stable version??", "equals", equals)
}
