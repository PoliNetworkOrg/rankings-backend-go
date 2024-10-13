package utils

import (
	"io/fs"
	"os"
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
