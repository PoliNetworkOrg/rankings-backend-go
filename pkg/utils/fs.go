package utils

import (
	"io/fs"
	"log/slog"
	"os"
	"path"
	"path/filepath"

	"github.com/PoliNetworkOrg/rankings-backend-go/pkg/constants"
)

func DoFolderExists(absPath string) (bool, error) {
	stat, err := os.Stat(absPath)

  switch err {
  case nil:
    return stat.IsDir(), nil

  case fs.ErrNotExist:
  return false, nil

  default: // other errors
  return false, err
  }
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
		return err
	}

	err = os.MkdirAll(path, os.ModePerm)
	return err
}

func CreateAllOutFolders(dataDir string) error {
	path1 := path.Join(dataDir, constants.OutputBaseFolder)
	path2 := path.Join(dataDir, constants.OutputHtmlFolder)

	err := CreateFolderIfNotExists(path1)
	if err != nil {
		return err
	}

	err = CreateFolderIfNotExists(path2)
	if err != nil {
		return err
	}

	return nil
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
