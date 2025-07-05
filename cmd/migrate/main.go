package main

import (
	"log/slog"
	"os"
	"path"
	"strings"

	"github.com/PoliNetworkOrg/rankings-backend-go/pkg/constants"
	"github.com/PoliNetworkOrg/rankings-backend-go/pkg/logger"
	"github.com/PoliNetworkOrg/rankings-backend-go/pkg/scraper"
	"github.com/PoliNetworkOrg/rankings-backend-go/pkg/utils"
	"github.com/PoliNetworkOrg/rankings-backend-go/pkg/writer"
)

func main() {
	slog.SetDefault(logger.GetDefaultLogger())
	opts := ParseOpts()
	htmlOutDir := path.Join(opts.dataDir, constants.OutputHtmlFolder) // abs path

	slog.Info("argv validation", "html_dir", opts.htmlDir, "data_dir", opts.dataDir, "html_out_dir", htmlOutDir)

	entries, err := utils.GetEntriesInFolder(opts.htmlDir)
	if err != nil {
		panic(err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		htmlInputPath := path.Join(opts.htmlDir, entry.Name())
		OrganizeHtml(htmlInputPath, htmlOutDir)
	}
}

func OrganizeHtml(inputPath string, outDir string) {
	entries, err := utils.GetEntriesInFolder(inputPath)
	if err != nil {
		panic(err)
	}

	var index []byte
	var byMerit, byCourse []scraper.HtmlPage = make([]scraper.HtmlPage, 0), make([]scraper.HtmlPage, 0)

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		fn := entry.Name()
		fp := path.Join(inputPath, fn)
		splitted := strings.Split(fn, "_")

		if strings.HasSuffix(fn, "_indice_M.html") {
			continue // merit index
		}
		if strings.HasSuffix(fn, "_sotto_indice.html") {
			continue // course index
		}
		if strings.HasSuffix(fn, "_generale.html") { // index
			data, err := os.ReadFile(fp)
			if err != nil {
				panic(err)
			}
			index = data
			continue
		}
		if len(splitted) >= 3 && splitted[2] == "sotto" { // course page
			data, err := os.ReadFile(fp)
			if err != nil {
				panic(err)
			}
			byCourse = append(byCourse, scraper.HtmlPage{Content: data, Id: fn})
			continue
		}
		if len(splitted) >= 5 && splitted[2] == "grad" && splitted[4] == "M" { // merit page
			data, err := os.ReadFile(fp)
			if err != nil {
				panic(err)
			}
			byMerit = append(byMerit, scraper.HtmlPage{Content: data, Id: fn})
			continue
		}
	}


	pathSplitted := strings.Split(inputPath, "/")
	id := pathSplitted[len(pathSplitted)-1]
	outRoot := path.Join(outDir, id)
	w, err := writer.NewWriter[[]byte](outRoot)
	if err != nil {
		panic(err)
	}

	if err := w.Write(constants.OutputHtmlRanking_IndexFilename, index); err != nil {
		slog.Error("Could not save ranking index html to filesystem", "ranking_url", inputPath)
		panic(err)
	}

	w.ChangeDirPath(path.Join(outRoot, constants.OutputHtmlRanking_ByMeritFolder))
	for _, page := range byMerit {
		err := w.Write(page.Id, page.Content)
		if err != nil {
			slog.Error("Could not save ranking byMerit table html to filesystem", "ranking_url", inputPath, "page_id", page.Id)
			panic(err)
		}
	}

	w.ChangeDirPath(path.Join(outRoot, constants.OutputHtmlRanking_ByCourseFolder))
	for _, page := range byCourse {
		err := w.Write(page.Id, page.Content)
		if err != nil {
			slog.Error("Could not save ranking byCourse table html to filesystem", "ranking_url", inputPath, "page_id", page.Id)
			panic(err)
		}
	}
}
