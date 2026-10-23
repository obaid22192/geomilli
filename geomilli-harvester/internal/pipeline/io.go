package pipeline

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

func EnsureDir(path string) error {
	return os.MkdirAll(path, 0o755)
}

func NewJSONLWriter(path string) (*os.File, *bufio.Writer, error) {
	if err := EnsureDir(filepath.Dir(path)); err != nil {
		return nil, nil, err
	}
	f, err := os.Create(path)
	if err != nil {
		return nil, nil, err
	}
	return f, bufio.NewWriterSize(f, 64*1024), nil
}

func WriteJSONL(w *bufio.Writer, v interface{}) error {
	payload, err := json.Marshal(v)
	if err != nil {
		return err
	}
	if _, err := w.Write(payload); err != nil {
		return err
	}
	if err := w.WriteByte('\n'); err != nil {
		return err
	}
	return nil
}

func MustClose(f *os.File) {
	if err := f.Close(); err != nil {
		fmt.Fprintf(os.Stderr, "close error: %v\n", err)
	}
}
