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
	"github.com/PoliNetworkOrg/rankings-backend-go/pkg/writer"
)

func main() {
	slog.SetDefault(logger.GetDefaultLogger())

	opts := ParseOpts()

	if opts.isTmpDir {
		slog.Warn("ATTENION! using tmp directory instead of data directory. Check --help for more information on data dir.", "dataDir", opts.dataDir)
	} else {
		slog.Info("Argv validation", "data_dir", opts.dataDir)
	}

	mansWriter, err := writer.NewWriter[[]scraper.Manifesto](opts.dataDir)
	mans := ScrapeManifestiWithLocal(&mansWriter, opts.force)
	if err != nil {
		panic(err)
	}

	slog.Info("finished scraping manifesti, writing to file...", "found", len(mans))

	err = mansWriter.JsonWrite(constants.OutputManifestiListFilename, mans, false)
	if err != nil {
		panic(err)
	}

	slog.Info("successfully written manifesti to file!")

	manEquals, err := DoLocalEqualsRemoteManifesti(&mansWriter)
	if err != nil {
		slog.Error("cannot perform comparison between local and remote versions", "err", err)
		return
	}

	slog.Info("Scrape manifesti, equals to remote version?? SUS", "equals", manEquals)
}

func ScrapeManifestiWithLocal(w *writer.Writer[[]scraper.Manifesto], force bool) []scraper.Manifesto {
	fn := constants.OutputManifestiListFilename
	fp := w.GetFilePath(fn)
	slog := slog.With("filepath", fp)

	if force {
		slog.Info("Scraping manifesti because of -f flag")
		return scraper.ScrapeManifesti(nil)
	}

	local, err := w.JsonRead(fn)
	if err != nil {
		switch {
		case errors.Is(err, os.ErrNotExist):
			slog.Info(fmt.Sprintf("%s file not found, running scraper...", fn))
		case errors.As(err, new(*json.SyntaxError)):
			slog.Error(fmt.Sprintf("%s contains malformed JSON, running scraper...", fn))
		case errors.As(err, new(*json.UnmarshalTypeError)):
			slog.Error(fmt.Sprintf("%s contains JSON not compatible with the Manifesto struct, running scraper...", fn))
		default:
			slog.Error("Failed to read from manifesti json file, running scraper...", "error", err)
		}
		return scraper.ScrapeManifesti(nil)
	}

	if len(local) == 0 {
		slog.Info(fmt.Sprintf("%s file is empty, running scraper...", fn))
		return scraper.ScrapeManifesti(nil)
	}

	slog.Info(fmt.Sprintf("loaded %d manifesti from %s json file, running scraper to check if there are new ones. If you would like to regenerate the whole thing, use the -f flag.", len(local), fn))
	return scraper.ScrapeManifesti(local)
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

	out := parser.ManifestiByDegreeType{}
	err = json.Unmarshal(bytes, &out.Data)
	if err != nil {
		return bytes, nil, err
	}

	return bytes, out.GetAll(), err
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
