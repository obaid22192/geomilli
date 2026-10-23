package runstate

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"sort"
	"time"
)

type Manifest struct {
	Job          string `json:"job"`
	InputPath    string `json:"input_path,omitempty"`
	OutputPath   string `json:"output_path,omitempty"`
	InputSHA256  string `json:"input_sha256,omitempty"`
	OutputSHA256 string `json:"output_sha256,omitempty"`
	Rows         int    `json:"rows,omitempty"`
	UpdatedAt    string `json:"updated_at"`
}

func WriteManifest(path string, m Manifest) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	m.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	b, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}

func FileSHA256(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func GlobSHA256(pattern string) (string, error) {
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return "", err
	}
	sort.Strings(matches)

	h := sha256.New()
	for _, match := range matches {
		info, err := os.Stat(match)
		if err != nil {
			return "", err
		}
		if info.IsDir() {
			continue
		}
		fileHash, err := FileSHA256(match)
		if err != nil {
			return "", err
		}
		if _, err := io.WriteString(h, match+":"+fileHash+"\n"); err != nil {
			return "", err
		}
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
