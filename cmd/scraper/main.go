package main

import (
	"io"
	"log/slog"
	"net/http"
	"net/url"

	"github.com/PoliNetworkOrg/rankings-backend-go/pkg/constants"
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

	if opts.isTmpDir {
		slog.Warn("ATTENION! using tmp directory instead of data directory. Check --help for more information on data dir.", "dataDir", opts.dataDir)
	} else {
		slog.Info("argv validation", "data_dir", opts.dataDir)
	}

	mans := ParseLocalOrScrapeManifesti(opts.dataDir, opts.force)
	manJson := writer.NewManifestiJson(mans)
	err = manJson.Write(opts.dataDir)
	if err != nil {
		panic(err)
	}

	manEquals, err := DoLocalEqualsRemoteManifesti(opts.dataDir)
	slog.Info("scrape manifesti, equals to remote version??", "equals", manEquals)
}

func ParseLocalOrScrapeManifesti(dataDir string, force bool) []scraper.Manifesto {
	if force {
		return scraper.ScrapeManifesti()
	}

	var mans []scraper.Manifesto
	mansB, err := writer.ReadManifestiJsonFile(dataDir)

	// bro this is nested
	if mansB == nil || err != nil {
		mans = scraper.ScrapeManifesti()
	} else {
		json, err := writer.ParseManifestiJson(mansB)
		if err != nil {
			mans = scraper.ScrapeManifesti()
		} else {
			mans = json.GetSlice()
		}
	}

	return mans
}

func DoLocalEqualsRemoteManifesti(dataDir string) (bool, error) {
	manBytes, err := writer.ReadManifestiJsonFile(dataDir)
	if err != nil {
		return false, err
	}
	remotePath, err := url.JoinPath(constants.WebGithubMainRawDataUrl, writer.ManifestiFilePath(""))
	if err != nil {
		return false, err
	}
	slog.Info("remote manifesti file", "url", remotePath)

	res, err := http.Get(remotePath)
	if err != nil {
		return false, err
	}
	defer res.Body.Close()
	remoteManBytes, err := io.ReadAll(res.Body)
	if err != nil {
		return false, err
	}
	return utils.TestJsonEquals(manBytes, remoteManBytes)
}
