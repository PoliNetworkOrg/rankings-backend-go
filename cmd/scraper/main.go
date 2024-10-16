package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"reflect"
	"slices"
	"strings"

	"github.com/PoliNetworkOrg/rankings-backend-go/pkg/constants"
	"github.com/PoliNetworkOrg/rankings-backend-go/pkg/logger"
	"github.com/PoliNetworkOrg/rankings-backend-go/pkg/parser"
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
		slog.Info("Argv validation", "data_dir", opts.dataDir)
	}

	mansWriter, err := writer.NewWriter[[]scraper.Manifesto](opts.dataDir)
	mans, scraped := ParseLocalOrScrapeManifesti(&mansWriter, opts.force)
	if err != nil {
		panic(err)
	}

	if scraped {
		slog.Info("scraped manifesti", "found", len(mans))

		err = mansWriter.JsonWrite(mans, constants.OutputManifestiListFilename, false)
		if err != nil {
			panic(err)
		}
	} else {
		slog.Info("parsed manifesti", "found", len(mans))
	}

	manEquals, err := DoLocalEqualsRemoteManifesti(&mansWriter)
	if err != nil {
		panic(err)
	}

	slog.Info("Scrape manifesti, equals to remote version??", "equals", manEquals)
}

func ParseLocalOrScrapeManifesti(w *writer.Writer[[]scraper.Manifesto], force bool) ([]scraper.Manifesto, bool) {
	fn := constants.OutputManifestiListFilename
	fp := w.GetFilePath(fn)
	slog := slog.With("filepath", fp)

	if force {
		slog.Info("Scraping manifesti because of -f flag")
		return scraper.ScrapeManifesti(), true
	}

	local, err := w.JsonRead(fn)
	if err != nil {
		switch {
		case errors.Is(err, os.ErrNotExist):
				slog.Info(fmt.Sprintf("%s file not found, running scraper...", fn))
			return scraper.ScrapeManifesti(), true
		case errors.As(err, new(*json.SyntaxError)):
			slog.Error(fmt.Sprintf("%s contains malformed JSON, running scraper...", fn))
			return scraper.ScrapeManifesti(), true
		case errors.As(err, new(*json.UnmarshalTypeError)):
			slog.Error(fmt.Sprintf("%s contains JSON not compatible with the Manifesto struct, running scraper...", fn))
			return scraper.ScrapeManifesti(), true
		default:
			slog.Error("Failed to read from manifesti json file, running scraper...", "error", err)
			return scraper.ScrapeManifesti(), true
		}
	}

	if len(local) == 0 {
		slog.Info(fmt.Sprintf("%s file is empty, running scraper...", fn))
		return scraper.ScrapeManifesti(), true
	}

	return local, false
}

func GetRemoteManifesti() ([]byte, []scraper.Manifesto, error) {
	remotePath, err := url.JoinPath(constants.WebGithubMainRawDataUrl, constants.OutputBaseFolder, constants.OutputManifestiListFilename)
	slog.Info("remote manifesti file", "url", remotePath)
	if err != nil {
		return nil, nil, err
	}

	res, err := http.Get(remotePath)
	if err != nil {
		return nil, nil, err
	}
	defer res.Body.Close()
	bytes, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, nil, err
	}

	out := parser.ManifestiJson{}
	err = json.Unmarshal(bytes, &out.Data)
	if err != nil {
		return bytes, nil, err
	}

	return bytes, out.GetSlice(), err
}

func DoLocalEqualsRemoteManifesti(w *writer.Writer[[]scraper.Manifesto]) (bool, error) {
	localSlice, err := w.JsonRead(constants.OutputManifestiListFilename)
	if err != nil {
		return false, err
	}

	_, remoteSlice, err := GetRemoteManifesti()
	if err != nil {
		return false, err
	}

	slices.SortStableFunc(localSlice, func(a, b scraper.Manifesto) int {
		name := strings.Compare(a.Name, b.Name)
		if name != 0 {
			return name
		}

		return strings.Compare(a.Location, b.Location)
	})
	slices.SortStableFunc(remoteSlice, func(a, b scraper.Manifesto) int {
		name := strings.Compare(a.Name, b.Name)
		if name != 0 {
			return name
		}

		return strings.Compare(a.Location, b.Location)
	})

	return reflect.DeepEqual(localSlice, remoteSlice), nil
}
