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
	"github.com/PoliNetworkOrg/rankings-backend-go/pkg/utils"
	"github.com/PoliNetworkOrg/rankings-backend-go/pkg/writer"
)

func main() {
	slog.SetDefault(logger.GetDefaultLogger())

	opts := ParseOpts()

	manifestiOutDir := opts.dataDir
	linksOutDir := path.Join(opts.dataDir, constants.OutputLinksFolder)
	bfLinksOutDir := path.Join(opts.dataDir, constants.OutputLinksFolder, constants.OutputBruteForceFolder)
	savedHtmlsFolder := path.Join(opts.dataDir, constants.OutputHtmlFolder)

	if opts.isTmpDir {
		slog.Warn("ATTENION! using tmp directory instead of data directory. Check --help for more information on data dir.", "dataDir", opts.dataDir)
	} else {
		slog.Info("Argv validation", "data_dir", opts.dataDir)
	}

	mansWriter := writer.NewWriter[[]scraper.Manifesto](manifestiOutDir)
	mans := scrapeManifestiWithLocal(&mansWriter, opts.force)

	slog.Info("finished scraping manifesti, writing to file...", "found", len(mans))

	err := mansWriter.JsonWrite(constants.OutputManifestiListFilename, mans, false)
	if err != nil {
		panic(err)
	}

	slog.Info("successfully written manifesti to file!")

	manEquals, err := doLocalEqualsRemoteManifesti(&mansWriter)
	if err != nil {
		slog.Error("cannot perform comparison between local and remote versions", "err", err)
		return
	}

	slog.Info("Scrape manifesti, equals to remote version??", "equals", manEquals)

	slog.Info("------------------------------------------")
	slog.Info("START scraping new rankings links")

	linksManager := scraper.NewLinksManager(linksOutDir)
	linksManager.PrintState("init")
	scrapedNewLinks := linksManager.FilterNewLinks(scraper.ScrapeRankingsLinks())

	bruteforceNewLinks := []string{}
	if opts.bruteforce.enabled {
		bruteforcer := scraper.NewBruteforcer(bfLinksOutDir, savedHtmlsFolder, opts.bruteforce.year)
		bruteforceNewLinks = linksManager.FilterNewLinks(bruteforcer.Start())
	}
	linksManager.PrintState("after bruteforce")

	scrapedLinks, brokenLinks := downloadHTMLs(utils.MergeUnique(scrapedNewLinks, bruteforceNewLinks), savedHtmlsFolder)
	linksManager.SetNewLinks(scrapedLinks, brokenLinks)
	linksManager.PrintState("after download HTMLs")

	linksManager.Write(opts.force)

	slog.Info("END scraping new rankings links", "scrapedCount", len(scrapedLinks), "brokenCount", len(brokenLinks))

	slog.Info("------------------------------------------")
}

func downloadHTMLs(newLinks []string, outDir string) ([]string, []string) {
	scrapedLinks := []string{}
	brokenLinks := []string{}

	if len(newLinks) == 0 {
		return scrapedLinks, brokenLinks
	}

	slog.Info("START Download new HTMLs", "newLinks", len(newLinks))

	downloadedCount := 0 // single html files downloaded count
	htmlRankings := scraper.DownloadRankings(newLinks)

	for _, r := range htmlRankings {
		if r.PageCount == 0 {
			// Politecnico loves to remove immediately the rankings from public availability, so they
			// might leave public the link in their "news" section, but they already removed the linked ranking (so stupid...)
			slog.Error("A ranking is empty. Probably its link is a 404.", "link", r.Url.String())
			brokenLinks = append(brokenLinks, r.Url.String())
			continue
		}

		root := path.Join(outDir, r.Id) // path of this ranking's html root folder
		w := writer.NewWriter[[]byte](root)

		if err := w.Write(constants.OutputHtmlRanking_IndexFilename, r.Index.Content); err != nil {
			slog.Error("Could not save ranking index html to filesystem", "ranking_url", r.Url.String())
			panic(err)
		}

		// update writer outDir path to byMerit folder
		w.ChangeDirPath(path.Join(root, constants.OutputHtmlRanking_ByMeritFolder))
		for _, page := range r.ByMerit {
			if err := w.Write(page.Id, page.Content); err != nil {
				slog.Error("Could not save ranking byMerit table html to filesystem", "ranking_url", r.Url.String(), "page_id", page.Id)
				panic(err)
			}
		}

		// update writer outDir path to byId folder
		w.ChangeDirPath(path.Join(root, constants.OutputHtmlRanking_ByIdFolder))
		for _, page := range r.ById {
			if err := w.Write(page.Id, page.Content); err != nil {
				slog.Error("Could not save ranking byId table html to filesystem", "ranking_url", r.Url.String(), "page_id", page.Id)
				panic(err)
			}
		}

		// update writer outDir path to byCourse folder
		w.ChangeDirPath(path.Join(root, constants.OutputHtmlRanking_ByCourseFolder))
		for _, page := range r.ByCourse {
			if err := w.Write(page.Id, page.Content); err != nil {
				slog.Error("Could not save ranking byCourse table html to filesystem", "ranking_url", r.Url.String(), "page_id", page.Id)
				panic(err)
			}
		}

		scrapedLinks = append(scrapedLinks, r.Url.String())
		downloadedCount += r.PageCount
	}

	slog.Info("END Download new HTMLs", "filesCount", downloadedCount)
	return scrapedLinks, brokenLinks
}

func scrapeManifestiWithLocal(w *writer.Writer[[]scraper.Manifesto], force bool) []scraper.Manifesto {
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

func doLocalEqualsRemoteManifesti(w *writer.Writer[[]scraper.Manifesto]) (bool, error) {
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
