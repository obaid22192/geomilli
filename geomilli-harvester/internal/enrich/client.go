package enrich

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"golang.org/x/time/rate"
)

type Client struct {
	http    *http.Client
	qps     float64
	limiter *rate.Limiter
}

func NewClient(timeout time.Duration, qps float64) *Client {
	if qps <= 0 {
		qps = 2
	}
	limiter := rate.NewLimiter(rate.Limit(qps), 1)
	return &Client{
		http:    &http.Client{Timeout: timeout},
		qps:     qps,
		limiter: limiter,
	}
}

func (c *Client) wait() {
	_ = c.limiter.Wait(context.Background())
}

func (c *Client) get(urlStr string) (*http.Response, error) {
	var resp *http.Response
	var err error
	maxRetries := 5
	backoff := 2 * time.Second

	for attempt := 0; attempt < maxRetries; attempt++ {
		c.wait()
		req, errReq := http.NewRequest(http.MethodGet, urlStr, nil)
		if errReq != nil {
			return nil, errReq
		}
		req.Header.Set("User-Agent", "TravayoAutoComplete/1.0 (contact@travayo.nl; +https://travayo.nl)")

		resp, err = c.http.Do(req)
		if err != nil {
			// Network or protocol error, sleep and retry
			time.Sleep(backoff)
			backoff *= 2
			continue
		}

		if resp.StatusCode == http.StatusTooManyRequests {
			// Respect Retry-After header if present
			retryAfter := resp.Header.Get("Retry-After")
			sleepDur := backoff
			if retryAfter != "" {
				if seconds, errParse := strconv.Atoi(retryAfter); errParse == nil {
					sleepDur = time.Duration(seconds) * time.Second
				}
			}
			resp.Body.Close()
			fmt.Printf("[enrich-client] received 429, sleeping for %v before retry (attempt %d/%d)\n", sleepDur, attempt+1, maxRetries)
			time.Sleep(sleepDur)
			backoff *= 2
			continue
		}

		return resp, nil
	}

	return resp, err
}

func (c *Client) FetchWikidataSummary(qid string) (int, string, error) {
	u := "https://www.wikidata.org/w/api.php?action=wbgetentities&format=json&props=sitelinks&ids=" + url.QueryEscape(qid)
	resp, err := c.get(u)
	if err != nil {
		return 0, "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return 0, "", fmt.Errorf("wikidata status=%d body=%s", resp.StatusCode, string(body))
	}
	var payload struct {
		Entities map[string]struct {
			Sitelinks map[string]struct {
				Title string `json:"title"`
			} `json:"sitelinks"`
		} `json:"entities"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return 0, "", err
	}
	entity, ok := payload.Entities[qid]
	if !ok {
		return 0, "", nil
	}
	sitelinks := len(entity.Sitelinks)
	enwiki := ""
	if v, ok := entity.Sitelinks["enwiki"]; ok {
		enwiki = v.Title
	}
	return sitelinks, enwiki, nil
}

func (c *Client) FetchPageviews(project string, articleTitle string) (int, error) {
	if project == "" || articleTitle == "" {
		return 0, nil
	}
	now := time.Now().UTC()
	start := now.AddDate(0, -12, 0).Format("2006010100")
	end := now.Format("2006010100")
	safeTitle := url.PathEscape(strings.ReplaceAll(articleTitle, " ", "_"))
	u := fmt.Sprintf("https://wikimedia.org/api/rest_v1/metrics/pageviews/per-article/%s/all-access/all-agents/%s/monthly/%s/%s", project, safeTitle, start, end)
	resp, err := c.get(u)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return 0, nil
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return 0, fmt.Errorf("pageviews status=%d body=%s", resp.StatusCode, string(body))
	}
	var payload struct {
		Items []struct {
			Views int `json:"views"`
		} `json:"items"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return 0, err
	}
	sum := 0
	for _, item := range payload.Items {
		sum += item.Views
	}
	return sum, nil
}
