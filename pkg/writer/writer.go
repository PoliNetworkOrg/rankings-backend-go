package writer

import (
	"encoding/json"
	"os"
	"path"

	"github.com/PoliNetworkOrg/rankings-backend-go/pkg/utils"
)

type Writer[T interface{}] struct {
	DirPath string
}

func NewWriter[T interface{}](dirPath string) (Writer[T], error) {
	err := utils.CreateFolderIfNotExists(dirPath)
	if err != nil {
		return Writer[T]{}, err
	}

	return Writer[T]{DirPath: dirPath}, nil
}

func (w *Writer[T]) getFilePath(filename string) string {
	return path.Join(w.DirPath, filename)
}

func (w *Writer[T]) Write(data []byte, filename string) error {
	p := w.getFilePath(filename)
	return os.WriteFile(p, data, 0664)
}

func (w *Writer[T]) Read(filename string) ([]byte, error) {
	p := w.getFilePath(filename)
	return os.ReadFile(p)
}

func (w *Writer[T]) JsonWrite(data T, filename string, indent bool) error {
	var bytes []byte
	var err error

	if indent {
		bytes, err = json.MarshalIndent(data, "", "	")
	} else {
		bytes, err = json.Marshal(data)
	}

	if err != nil {
		return err
	}

	return w.Write(bytes, filename)
}

func (w *Writer[T]) JsonRead(filename string) (T, error) {
	var out T
	bytes, err := w.Read(filename)
	if err != nil {
		return out, err
	}

	err = json.Unmarshal(bytes, &out)
	if err != nil {
		return out, err
	}

	return out, nil
}
