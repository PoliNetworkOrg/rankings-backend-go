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
	isTmpDir bool
}

func ParseOpts() Opts {
	tmpDir, _ := utils.TmpDirectory() // we don't care if err

	// definition
	help := getopt.BoolLong("help", 'h', "Shows the help menu")
	dataDir := getopt.StringLong("data-dir", 'd', tmpDir, "Path of the data folder (containing html, json, ...). Defaults to tmp directory")

	// parsing
	getopt.Parse()

	if *help {
		getopt.Usage()
		os.Exit(0)
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
		isTmpDir: absDataDir == tmpDir,
	}
}
