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
	outDir := path.Join(opts.dataDir, constants.OutputBaseFolder, constants.OutputParsedManifestiFolder) // abs path

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
	dtmWriter, err := writer.NewWriter[parser.ManifestiByDegreeType](outDir)
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

		slog.Info("manifesti parser: successful write", "filename", fn)
	}

	byCourseMans := parser.ParseManifestiByCourse(inputMans)
	cmWriter, err := writer.NewWriter[parser.ManifestiByCourse](outDir)
	if err != nil {
		panic(err)
	}

	cmFn := constants.OutputParsedManifestiAllFilename
	err = cmWriter.JsonWrite("all.json", byCourseMans, false)
	if err != nil {
		slog.Error("error while writing parsed manifesti byCourse (all)", "filename", cmFn)
		panic(err)
	}

	slog.Info("manifesti parser: successful write", "filename", cmFn)
}
