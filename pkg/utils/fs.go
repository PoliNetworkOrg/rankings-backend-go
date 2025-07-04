package utils

import (
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

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

func ReadAllFilesInFolder(absPath string) ([][]byte, error) {
	path := absPath
	if !filepath.IsAbs(path) {
		slog.Warn("asking for absPath, provided a relative path", "provided", absPath)
		var err error
		path, err = filepath.Abs(absPath)
		if err != nil {
			return nil, err
		}
	}

	exists, err := DoFolderExists(path)
	if !exists || err != nil {
		return nil, fmt.Errorf("Folder does not exist. Eventual error: %w", err)
	}

	entries, err := os.ReadDir(absPath)
	if err != nil {
		return nil, err
	}

	htmls := make([][]byte, 0)
	for _, file := range entries {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".html") {
			filePath := filepath.Join(absPath, file.Name())
			content, err := os.ReadFile(filePath)
			if err != nil {
				return nil, fmt.Errorf("failed to read file %s: %w", filePath, err)
			}
			htmls = append(htmls, content)
		}
	}

	return htmls, nil
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

func MakeFilename(str string, ext string) string {
	str = strings.TrimSpace(str)
	str = strings.ToLower(str)
	str = strings.ReplaceAll(str, " ", "_")
	return str + ext
}
