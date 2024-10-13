package main

import (
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/lmittmann/tint"
	"github.com/pborman/getopt/v2"
)

type Opts struct {
	dataDir string
}

func doFolderExists(path string) (bool, error) {
	stat, err := os.Stat(path)

	if err == nil {
		return stat.IsDir(), nil
	}

	if os.IsNotExist(err) {
		return false, nil
	}

	return false, err
}

func parseOpts() Opts {
	opts := Opts{}

	// definition
	help := getopt.BoolLong("help", 'h', "Shows the help menu")
	dataDir := getopt.StringLong("data-dir", 'd', "", "Path of the data folder (containing html, json, ...)")

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

	dataDirExists, err := doFolderExists(opts.dataDir)

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

func main() {
	slog.SetDefault(
		slog.New(
			tint.NewHandler(os.Stderr, &tint.Options{
				Level:      slog.LevelDebug,
				TimeFormat: time.Kitchen,
			}),
		))

	opts := parseOpts()

	slog.Info("argv validation", "data_dir", opts.dataDir)
}
