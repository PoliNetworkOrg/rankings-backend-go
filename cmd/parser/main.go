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
	checkPhasesOutDir := path.Join(opts.dataDir, constants.OutputBaseFolder, "test")                              // abs path
	idHashIndexOutDir := path.Join(opts.dataDir, constants.OutputBaseFolder)                                      // abs path

	slog.Info("argv validation", "data_dir", opts.dataDir)

	smWriter, err := writer.NewWriter[[]scraper.Manifesto](opts.dataDir)
	if err != nil {
		panic(err)
	}

	inputMans, err := smWriter.JsonRead(constants.OutputManifestiListFilename)
	if err != nil {
		panic(err)
	}

	byDegTypeMans := parser.ParseManifestiByDegreeType(inputMans)
	dtmWriter, err := writer.NewWriter[parser.ManifestiByDegreeType](manifestiOutDir)
	if err != nil {
		panic(err)
	}
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
	cmWriter, err := writer.NewWriter[parser.ManifestiByCourse](manifestiOutDir)
	if err != nil {
		panic(err)
	}

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
	rankingWriter, err := writer.NewWriter[parser.Ranking](rankingsOutDir)
	if err != nil {
		panic(err)
	}

	checkPhases := parser.NewCheckPhases(checkPhasesOutDir)
	idHashIndexParser := parser.NewIdHashIndexParser(idHashIndexOutDir)
	for _, entry := range htmlFolders {
		if !entry.IsDir() {
			continue
		}

		id := entry.Name()
		if id == "style" {
			slog.Warn("[rankings] skipping html 'style' folder")
			continue
		}

		rp, err := parser.NewRankingParser(path.Join(opts.dataDir, constants.OutputHtmlFolder, id))
		if err != nil {
			panic(err)
		}

		ranking := rp.Parse()
		if ranking == nil {
			slog.Error("[rankings] could not parse. return nil", "id", id)
			continue
		}
		checkPhases.Add(ranking)

		err = rankingWriter.JsonWrite(id+".json", *ranking, true)
		idHashIndexParser.Add(ranking)

		err = rankingWriter.JsonWrite(id+".json", *ranking, true)
		if err != nil {
			slog.Error("[rankings] error while writing to fs (PANIC)", "id", id)
			panic(err)
		}

		slog.Info("[rankings] successful write", "id", id)
	}

	if err = checkPhases.Write(); err != nil {
		slog.Error("could not write checkPhases.", "error", err)
	}

	if err = idHashIndexParser.Write(); err != nil {
		slog.Error("could not write studentIdHashIndex.", "error", err)
	}
}
