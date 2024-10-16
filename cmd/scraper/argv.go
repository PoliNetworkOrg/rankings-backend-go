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
	dataDir string
}

func ParseOpts() Opts {
	tmpDir, _ := utils.TmpDirectory() // we don't care if err

	opts := Opts{}

	// definition
	help := getopt.BoolLong("help", 'h', "Shows the help menu")
	dataDir := getopt.StringLong("data-dir", 'd', tmpDir, "Path of the data folder (containing html, json, ...)")

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
	opts.dataDir = absDataDir

	dataDirExists, err := utils.DoFolderExists(opts.dataDir)

	if !dataDirExists {
		slog.Error("You must set the --data-dir flag to an existing directory.")
		os.Exit(2)
	}
	if err != nil {
		tint.Err(err)
		os.Exit(1)
	}

	return opts
}
