package utils

import (
	"io/fs"
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

func CreateAllOutFolders(dataDir string) error {
	path1 := path.Join(dataDir, constants.OutputBaseFolder)
	path2 := path.Join(dataDir, constants.OutputHtmlFolder)

	err := os.MkdirAll(path1, os.ModePerm)
	if err != nil {
		return err
	}

	err = os.MkdirAll(path2, os.ModePerm)
	if err != nil {
		return err
	}

	return nil
}

func TmpDirectory() (string, error) {
	fullPath, err := filepath.Abs(constants.TmpDirectoryName)
	if err != nil {
		return "", err
	}

	exists, err := DoFolderExists(constants.TmpDirectoryName)
	if err != nil {
		return "", err
	}

	if !exists {
		os.Mkdir(fullPath, os.ModePerm)
	}

	return fullPath, nil
}
