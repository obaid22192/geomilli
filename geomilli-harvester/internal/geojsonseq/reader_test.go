package geojsonseq_test

import (
	"bytes"
	"io"
	"testing"

	"github.com/travayo/auto-complete/internal/geojsonseq"
)

func TestReaderNext(t *testing.T) {
	// RFC 8142 records separated by RS byte (0x1e) and newline
	data := []byte("\x1e{\"type\":\"Feature\",\"id\":\"node/123\",\"properties\":{\"name\":\"Brussels\"},\"geometry\":{\"type\":\"Point\",\"coordinates\":[4.3517,50.8503]}}\n" +
		"\n" + // empty line to test skipping
		"\x1e{\"type\":\"Feature\",\"id\":\"way/456\",\"properties\":{\"name\":\"Grand Place\"},\"geometry\":{\"type\":\"Point\",\"coordinates\":[4.3524,50.8467]}}\n")

	reader := geojsonseq.NewReader(bytes.NewReader(data))

	// First record
	f1, err := reader.Next()
	if err != nil {
		t.Fatalf("unexpected error on first record: %v", err)
	}
	if f1.ID != "node/123" {
		t.Errorf("expected ID 'node/123', got '%v'", f1.ID)
	}
	if f1.Properties["name"] != "Brussels" {
		t.Errorf("expected name 'Brussels', got '%v'", f1.Properties["name"])
	}

	// Second record
	f2, err := reader.Next()
	if err != nil {
		t.Fatalf("unexpected error on second record: %v", err)
	}
	if f2.ID != "way/456" {
		t.Errorf("expected ID 'way/456', got '%v'", f2.ID)
	}
	if f2.Properties["name"] != "Grand Place" {
		t.Errorf("expected name 'Grand Place', got '%v'", f2.Properties["name"])
	}

	// EOF
	f3, err := reader.Next()
	if err != io.EOF {
		t.Errorf("expected io.EOF at end of stream, got f=%v, err=%v", f3, err)
	}
}
