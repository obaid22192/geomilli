package meili

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type QueryExpectation struct {
	Query          string
	ExpectedTokens []string
}

func (i *Indexer) SearchHits(indexUID string, query string, limit int) ([]map[string]interface{}, error) {
	if limit <= 0 {
		limit = 10
	}
	payload := map[string]interface{}{"q": query, "limit": limit}
	b, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(http.MethodPost, i.host+"/indexes/"+indexUID+"/search", bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if i.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+i.apiKey)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("search status=%d body=%s", resp.StatusCode, string(body))
	}
	var parsed struct {
		Hits []map[string]interface{} `json:"hits"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, err
	}
	return parsed.Hits, nil
}

func foldDiacritics(s string) string {
	s = strings.ToLower(s)
	replacer := strings.NewReplacer(
		"é", "e", "è", "e", "ê", "e", "ë", "e",
		"à", "a", "á", "a", "â", "a", "ä", "a",
		"î", "i", "ï", "i", "í", "i",
		"ô", "o", "ö", "o", "ó", "o",
		"ù", "u", "û", "u", "ü", "u", "ú", "u",
		"ç", "c", "ñ", "n",
	)
	return replacer.Replace(s)
}

func (i *Indexer) ValidateGoldenQueries(indexUID string, checks []QueryExpectation, limit int) error {
	for _, check := range checks {
		hits, err := i.SearchHits(indexUID, check.Query, limit)
		if err != nil {
			return fmt.Errorf("query %q failed: %w", check.Query, err)
		}
		if len(hits) == 0 {
			return fmt.Errorf("query %q returned no hits", check.Query)
		}

		haystacks := make([]string, 0, len(hits))
		for _, hit := range hits {
			b, _ := json.Marshal(hit)
			haystacks = append(haystacks, foldDiacritics(string(b)))
		}

		if len(check.ExpectedTokens) == 0 {
			continue
		}

		matched := false
		for _, token := range check.ExpectedTokens {
			needle := foldDiacritics(strings.TrimSpace(token))
			if needle == "" {
				continue
			}
			for _, hs := range haystacks {
				if strings.Contains(hs, needle) {
					matched = true
					break
				}
			}
			if matched {
				break
			}
		}

		if !matched {
			return fmt.Errorf("query %q missing any expected token %v in top %d hits", check.Query, check.ExpectedTokens, limit)
		}
	}
	return nil
}
