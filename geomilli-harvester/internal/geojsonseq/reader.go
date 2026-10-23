package geojsonseq

import (
	"bufio"
	"encoding/json"
	"io"
)

// Feature is a minimal GeoJSON feature structure for osmium geojsonseq exports.
type Feature struct {
	Type       string                 `json:"type"`
	ID         interface{}            `json:"id"`
	Properties map[string]interface{} `json:"properties"`
	Geometry   Geometry               `json:"geometry"`
	BBox       []float64              `json:"bbox,omitempty"`
}

type Geometry struct {
	Type        string      `json:"type"`
	Coordinates interface{} `json:"coordinates"`
}

// Reader streams one geojsonseq feature per line using bounded memory.
// The internal bufio.Reader size is 4 MB to handle large boundary polygons.
type Reader struct {
	r *bufio.Reader
}

func NewReader(src io.Reader) *Reader {
	return &Reader{r: bufio.NewReaderSize(src, 4*1024*1024)}
}

// rsSep is the RFC 8142 record separator byte osmium prepends to every feature.
const rsSep = 0x1e

func (r *Reader) Next() (*Feature, error) {
	for {
		line, err := r.r.ReadBytes('\n')
		if err != nil && len(line) == 0 {
			if err == io.EOF {
				return nil, io.EOF
			}
			return nil, err
		}
		// Strip leading RS byte and surrounding whitespace before parsing.
		line = trimRS(line)
		if len(line) == 0 {
			if err == io.EOF {
				return nil, io.EOF
			}
			continue
		}
		f := &Feature{}
		if uErr := json.Unmarshal(line, f); uErr != nil {
			if err == io.EOF {
				return nil, io.EOF
			}
			return nil, uErr
		}
		return f, nil
	}
}

// trimRS strips the RFC 8142 record separator (0x1e) and surrounding whitespace/newlines.
func trimRS(b []byte) []byte {
	if len(b) > 0 && b[0] == rsSep {
		b = b[1:]
	}
	// trim trailing \r\n and spaces
	for len(b) > 0 && (b[len(b)-1] == '\n' || b[len(b)-1] == '\r' || b[len(b)-1] == ' ') {
		b = b[:len(b)-1]
	}
	// trim leading whitespace
	for len(b) > 0 && (b[0] == '\n' || b[0] == '\r' || b[0] == ' ') {
		b = b[1:]
	}
	return b
}
