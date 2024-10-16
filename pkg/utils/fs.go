package utils

import (
	"errors"
	"io/fs"
	"log/slog"
	"os"
	"path"
	"path/filepath"

	"github.com/PoliNetworkOrg/rankings-backend-go/pkg/constants"
)

func DoFolderExists(absPath string) (bool, error) {
	stat, err := os.Stat(absPath)

	if err != nil {
		switch {
		case errors.Is(err, fs.ErrNotExist):
			return false, nil
		default:
			return false, err
		}
	}

	return stat.IsDir(), nil
}

func CreateFolderIfNotExists(absPath string) error {
	path := absPath
	if !filepath.IsAbs(path) {
		slog.Warn("asking for absPath, provided a relative path", "provided", absPath)
		var err error
		path, err = filepath.Abs(absPath)
		if err != nil {
			return err 
		}
	}

	exists, err := DoFolderExists(path)
	if err != nil {
		return err
	}

	if exists {
		return nil
	}

	err = os.MkdirAll(path, os.ModePerm)
	return err
}

func TmpDirectory() (string, error) {
	tmpPath, err := filepath.Abs(constants.TmpDirectoryName)
	if err != nil {
		return "", err
	}

	err = CreateFolderIfNotExists(tmpPath)
	if err != nil {
		return "", err
	}

	return tmpPath, nil
}
