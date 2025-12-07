package main

import (
	"log/slog"
	"path"

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
	manifestiOutDir := path.Join(opts.dataDir, constants.OutputBaseFolder, constants.OutputParsedManifestiFolder) // abs path
	rankingsOutDir := path.Join(opts.dataDir, constants.OutputBaseFolder, constants.OutputParsedRankingsFolder)   // abs path
	indexesOutDir := path.Join(opts.dataDir, constants.OutputBaseFolder, constants.OutputIndexesFolder)           // abs path

	slog.Info("argv validation", "data_dir", opts.dataDir)

	smWriter := writer.NewWriter[[]scraper.Manifesto](opts.dataDir)

	inputMans, err := smWriter.JsonRead(constants.OutputManifestiListFilename)
	if err != nil {
		panic(err)
	}

	byDegTypeMans := parser.ParseManifestiByDegreeType(inputMans)
	dtmWriter := writer.NewWriter[parser.ManifestiByDegreeType](manifestiOutDir)

	for _, m := range byDegTypeMans {
		fn := utils.MakeFilename(m.DegreeType, ".json")
		err := dtmWriter.JsonWrite(fn, m, false)
		if err != nil {
			slog.Error("error while writing parsed manifesti byDegreeType (grouped)", "filename", fn)
			panic(err)
		}

		slog.Info("[manifesti] successful write", "filename", fn)
	}

	byCourseMans := parser.ParseManifestiByCourse(inputMans)
	cmWriter := writer.NewWriter[parser.ManifestiByCourse](manifestiOutDir)

	cmFn := constants.OutputParsedManifestiAllFilename
	err = cmWriter.JsonWrite(cmFn, byCourseMans, false)
	if err != nil {
		slog.Error("error while writing parsed manifesti byCourse (all)", "filename", cmFn)
		panic(err)
	}

	slog.Info("[manifesti] successful write", "filename", cmFn)

	htmlFolderPath := path.Join(opts.dataDir, constants.OutputHtmlFolder)
	htmlFolders, err := utils.GetEntriesInFolder(htmlFolderPath)
	if err != nil {
		slog.Error("error while listing saved html folders", "path", htmlFolderPath)
		panic(err)
	}

	// note: this is hardcoded for testing
	rankingWriter := writer.NewWriter[parser.Ranking](rankingsOutDir)

	indexGenerator := parser.NewIndexGenerator(indexesOutDir)

	idHashIndexParser := parser.NewIdHashIndexParser(indexesOutDir)
	for _, entry := range htmlFolders {
		if !entry.IsDir() {
			continue
		}

		id := entry.Name()
		if id == "style" {
			slog.Warn("[rankings] skipping html 'style' folder")
			continue
		}

		rp := parser.NewRankingParser(path.Join(opts.dataDir, constants.OutputHtmlFolder, id))

		ranking := rp.Parse()
		if ranking == nil {
			slog.Error("[rankings] could not parse. return nil", "id", id)
			continue
		}
		indexGenerator.Add(ranking)

		err = rankingWriter.JsonWrite(id+".json", *ranking, true)
		idHashIndexParser.Add(ranking)

		err = rankingWriter.JsonWrite(id+".json", *ranking, true)
		if err != nil {
			slog.Error("[rankings] error while writing to fs (PANIC)", "id", id)
			panic(err)
		}

		slog.Info("[rankings] successful write", "id", id)
	}

	if err = indexGenerator.Generate(); err != nil {
		slog.Error("could not write indexes.", "error", err)
	}

	if err = idHashIndexParser.Write(); err != nil {
		slog.Error("could not write studentIdHashIndex.", "error", err)
	}
}
