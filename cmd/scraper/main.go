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
	"path"
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
	if err != nil {
		panic(err)
	}
	mans := ScrapeManifestiWithLocal(&mansWriter, opts.force)

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

	slog.Info("Scrape manifesti, equals to remote version??", "equals", manEquals)

	slog.Info("------------------------------------------")
	slog.Info("START scraping new rankings links")
	rankingsLinksWriter, err := writer.NewWriter[[]string](opts.dataDir)
	if err != nil {
		panic(err)
	}
	newRankingsLinks := scrapeRankingsLinks(&rankingsLinksWriter)
	slog.Info("END scraping new rankings links", "count", len(newRankingsLinks))
	slog.Info("------------------------------------------")
	downloadedCount := 0
	if len(newRankingsLinks) > 0 {
		slog.Info("START downloading new rankings")

		htmlRankings := scraper.DownloadRankings(newRankingsLinks)
		successUrls := make([]string, 0)
		for _, r := range htmlRankings {
			successUrls = append(successUrls, r.Url.String())
			if len(r.Pages) == 0 {
				// we add also these ones to the successUrls, because those links are already expired.
				// Politecnico loves to remove immediately the rankings from public availability, so they
				// might leave public the link in their "news" section, but they already removed the linked ranking (so stupid...)
				slog.Error("A ranking is empty. Probably its link is a 404.", "link", r.Url.String())
				continue
			}

			rankingsHtmlWriter, err := writer.NewWriter[[]byte](path.Join(opts.dataDir, constants.OutputHtmlFolder, r.Id))
			if err != nil {
				panic(err)
			}

			downloadedCount += len(r.Pages)
			for _, page := range r.Pages {
				filename := page.Id + ".html"
				err := rankingsHtmlWriter.Write(filename, page.Content)
				if err != nil {
					slog.Error("Could not save html to filesystem")
					panic(err)
				}
			}
		}

		err = rankingsLinksWriter.AppendLines(constants.OutputLinksFilename, successUrls)
		if err != nil {
			slog.Error("Could not append new rankings links to file")
			panic(err)
		}
	}

	slog.Info("END", "downloaded page count", downloadedCount)
	slog.Info("------------------------------------------")
}

func scrapeRankingsLinks(w *writer.Writer[[]string]) []string {
	fn := constants.OutputLinksFilename
	fp := w.GetFilePath(fn)
	slog := slog.With("filepath", fp)

	savedLinks, err := w.ReadLines(fn)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			slog.Warn("Saved file not found, running scraper...")
			savedLinks = make([]string, 0)
		} else {
			slog.Error("Could not read lines from saved rankings links file. SCRAPER SKIPPED", "error", err)
			return nil
		}
	} else {
		slog.Info("already saved rankings links", "count", len(savedLinks))
	}

	newLinks := scraper.ScrapeRankingsLinks(savedLinks)
	return newLinks
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
	remotePath, err := url.JoinPath(constants.WebGithubMainRawDataUrl, constants.OutputBaseFolder, "manifesti.json") // this is still the old filename
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

	out := parser.RemoteManifesti{}
	err = json.Unmarshal(bytes, &out.Data)
	if err != nil {
		return bytes, nil, err
	}

	return bytes, out.ToList(), err
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

	if len(localSlice) != len(remoteSlice) {
		return false, nil
	}

	// Politecnico loves changing domains and web servers
	// this is to not false our check
	for i := range len(localSlice) {
		rUrl, err := url.Parse(remoteSlice[i].Url)
		if err != nil {
			return false, err
		}
		lUrl, err := url.Parse(localSlice[i].Url)
		if err != nil {
			return false, err
		}

		lUrl.Host = "DOMAIN.polimi.it"
		rUrl.Host = "DOMAIN.polimi.it"

		localSlice[i].Url = lUrl.String()
		remoteSlice[i].Url = rUrl.String()
	}

	return reflect.DeepEqual(localSlice, remoteSlice), nil
}
