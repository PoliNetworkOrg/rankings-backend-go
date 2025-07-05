package main

import (
	"log/slog"
	"os"
	"path/filepath"

	"github.com/PoliNetworkOrg/rankings-backend-go/pkg/utils"
	"github.com/lmittmann/tint"
	"github.com/pborman/getopt/v2"
)

type Opts struct {
	dataDir  string
	htmlDir  string
	isTmpDir bool
}

func ParseOpts() Opts {
	tmpDir, _ := utils.TmpDirectory() // we don't care if err

	// definition
	help := getopt.BoolLong("help", 'h', "Shows the help menu")
	htmlDir := getopt.StringLong("html-dir", 'i', "", "Path of the folder containing the old html files.")
	dataDir := getopt.StringLong("data-dir", 'o', tmpDir, "Path of the new data folder (containing html, json, ...). Defaults to tmp directory")

	// parsing
	getopt.Parse()

	if *help {
		getopt.Usage()
		os.Exit(0)
	}

	if *htmlDir == "" {
		slog.Error("You must set the --html-dir flag.")
		os.Exit(2)
	}

	absHtmlDir, err := filepath.Abs(*htmlDir)
	if err != nil {
		tint.Err(err)
		os.Exit(1)
	}

	absDataDir, err := filepath.Abs(*dataDir)
	if err != nil {
		tint.Err(err)
		os.Exit(1)
	}

	dataDirExists, err := utils.DoFolderExists(absDataDir)
	if !dataDirExists {
		slog.Error("You must set the --data-dir flag to an existing directory.")
		os.Exit(2)
	}
	if err != nil {
		tint.Err(err)
		os.Exit(1)
	}

	return Opts{
		dataDir:  absDataDir,
		htmlDir:  absHtmlDir,
		isTmpDir: absDataDir == tmpDir,
	}
}
