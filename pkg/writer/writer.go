package writer

import (
	"bufio"
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

func (w *Writer[T]) GetFilePath(filename string) string {
	return path.Join(w.DirPath, filename)
}

func (w *Writer[T]) Write(filename string, data []byte) error {
	p := w.GetFilePath(filename)
	return os.WriteFile(p, data, 0664)
}

func (w *Writer[T]) Read(filename string) ([]byte, error) {
	p := w.GetFilePath(filename)
	return os.ReadFile(p)
}

func (w *Writer[T]) ReadLines(filename string) ([]string, error) {
	p := w.GetFilePath(filename)
	f, err := os.Open(p)
	if err != nil {
		return nil, err
	}

	defer f.Close()
	lines := make([]string, 0)
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines, nil
}

func (w *Writer[T]) WriteLines(filename string, data []string) error {
	p := w.GetFilePath(filename)
	f, err := os.OpenFile(p, os.O_WRONLY|os.O_CREATE, 0664)
	if err != nil {
		return err
	}

	defer f.Close()
	writer := bufio.NewWriter(f)
	for _, line := range data { 
		_, err := writer.WriteString(line + "\n") 
		if err != nil {
			return err
		}
	}
	return writer.Flush()
}

func (w *Writer[T]) AppendLines(filename string, data []string) error {
	p := w.GetFilePath(filename)
	f, err := os.OpenFile(p, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0664)
	if err != nil {
		return err
	}

	defer f.Close()
	writer := bufio.NewWriter(f)
	for _, line := range data { 
		_, err := writer.WriteString(line + "\n") 
		if err != nil {
			return err
		}
	}

	return writer.Flush()
}

func (w *Writer[T]) JsonWrite(filename string, data T, indent bool) error {
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

	return w.Write(filename, bytes)
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
