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
	err := utils.CreateAllOutFolders(opts.dataDir)
	if err != nil {
		slog.Error("Cannot create output folder(s)", "error", err)
	}

	tmpDir, err := utils.TmpDirectory()
	if err != nil {
		panic(err)
	}

	if opts.dataDir == tmpDir {
		slog.Warn("ATTENION! using tmp directory instead of data directory. Check --help for more information on data dir.", "dataDir", opts.dataDir)
	} else {
		slog.Info("argv validation", "data_dir", opts.dataDir)
	}


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
