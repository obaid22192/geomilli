package taxonomy

import (
	"context"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/geomilli/geomilli-mcp/internal/apiclient"
)

// Default 19 canonical classes per config/taxonomy.yaml for cold-cache fallback.
var defaultClasses = []string{
	"Airport",
	"Beach",
	"BusStation",
	"City",
	"Country",
	"Entertainment",
	"FerryTerminal",
	"HistoricSite",
	"Hotel",
	"Island",
	"NaturalFeature",
	"Neighborhood",
	"Park",
	"Region",
	"Restaurant",
	"Shopping",
	"Stadium",
	"TouristAttraction",
	"TrainStation",
}

// Default language whitelist codes for cold-cache fallback.
var defaultLanguages = []string{
	"ar", "de", "en", "es", "fr", "it", "ja", "nl", "pt", "ru", "zh",
}

// Fetcher defines the upstream API methods needed by the taxonomy cache.
type Fetcher interface {
	GetTaxonomy(ctx context.Context) (*apiclient.TaxonomyConfig, error)
	GetLanguages(ctx context.Context) (*apiclient.LanguagesConfig, error)
}

// Cache provides thread-safe, TTL-cached taxonomy and language metadata with cold-cache fallbacks.
type Cache struct {
	fetcher Fetcher
	ttl     time.Duration
	mu      sync.RWMutex

	lastFetched   time.Time
	classNames    []string
	subclasses    map[string][]string // class -> list of subclasses
	allSubclasses []string
	languageCodes []string
	taxonomy      *apiclient.TaxonomyConfig
	languages     *apiclient.LanguagesConfig
}

// NewCache initializes a new Taxonomy Cache with cold-cache fallback defaults.
func NewCache(fetcher Fetcher, ttl time.Duration) *Cache {
	c := &Cache{
		fetcher:       fetcher,
		ttl:           ttl,
		classNames:    append([]string(nil), defaultClasses...),
		subclasses:    make(map[string][]string),
		allSubclasses: []string{},
		languageCodes: append([]string(nil), defaultLanguages...),
	}
	sort.Strings(c.classNames)
	sort.Strings(c.languageCodes)
	return c
}

// Refresh fetches fresh taxonomy and language configuration from the upstream API.
func (c *Cache) Refresh(ctx context.Context) error {
	tax, errTax := c.fetcher.GetTaxonomy(ctx)
	lang, errLang := c.fetcher.GetLanguages(ctx)

	if errTax != nil && errLang != nil {
		// Retain existing/cold cache on failure
		return errTax
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if errTax == nil && tax != nil {
		c.taxonomy = tax
		classSet := make(map[string]struct{})
		subclassMap := make(map[string][]string)
		allSubSet := make(map[string]struct{})

		for _, cd := range tax.Classes {
			cName := strings.TrimSpace(cd.Name)
			if cName == "" {
				continue
			}
			classSet[cName] = struct{}{}

			subSetForClass := make(map[string]struct{})
			for _, m := range cd.Match {
				for _, v := range m.Values {
					val := strings.TrimSpace(v)
					if val != "" && val != "*" {
						subSetForClass[val] = struct{}{}
						allSubSet[val] = struct{}{}
					}
				}
			}

			subs := make([]string, 0, len(subSetForClass))
			for s := range subSetForClass {
				subs = append(subs, s)
			}
			sort.Strings(subs)
			subclassMap[cName] = subs
		}

		classNames := make([]string, 0, len(classSet))
		for k := range classSet {
			classNames = append(classNames, k)
		}
		sort.Strings(classNames)
		if len(classNames) > 0 {
			c.classNames = classNames
			c.subclasses = subclassMap

			allSubs := make([]string, 0, len(allSubSet))
			for s := range allSubSet {
				allSubs = append(allSubs, s)
			}
			sort.Strings(allSubs)
			c.allSubclasses = allSubs
		}
	}

	if errLang == nil && lang != nil {
		c.languages = lang
		var langCodes []string
		for _, item := range lang.Languages {
			code := strings.ToLower(strings.TrimSpace(item.Code))
			if code != "" && item.Enabled {
				langCodes = append(langCodes, code)
			}
		}
		if len(langCodes) > 0 {
			sort.Strings(langCodes)
			c.languageCodes = langCodes
		}
	}

	c.lastFetched = time.Now()
	return nil
}

// RefreshIfNeeded refreshes the cache if the TTL has expired or if never fetched.
func (c *Cache) RefreshIfNeeded(ctx context.Context) error {
	c.mu.RLock()
	expired := time.Since(c.lastFetched) >= c.ttl
	c.mu.RUnlock()

	if expired {
		return c.Refresh(ctx)
	}
	return nil
}

// GetClassNames returns the sorted list of valid taxonomy class names.
func (c *Cache) GetClassNames() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	res := make([]string, len(c.classNames))
	copy(res, c.classNames)
	return res
}

// GetSubclasses returns subclasses associated with a specific class, if known.
func (c *Cache) GetSubclasses(className string) []string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	subs, ok := c.subclasses[className]
	if !ok {
		return nil
	}
	res := make([]string, len(subs))
	copy(res, subs)
	return res
}

// GetAllSubclasses returns all known subclasses across all classes.
func (c *Cache) GetAllSubclasses() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	res := make([]string, len(c.allSubclasses))
	copy(res, c.allSubclasses)
	return res
}

// GetLanguageCodes returns the sorted whitelist of supported language codes.
func (c *Cache) GetLanguageCodes() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	res := make([]string, len(c.languageCodes))
	copy(res, c.languageCodes)
	return res
}

// IsValidClass checks whether a class name matches a valid taxonomy class (case-insensitive).
func (c *Cache) IsValidClass(className string) (string, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	lower := strings.ToLower(strings.TrimSpace(className))
	for _, cn := range c.classNames {
		if strings.ToLower(cn) == lower {
			return cn, true
		}
	}
	return "", false
}

// IsValidLanguage checks whether a language code is in the whitelist.
func (c *Cache) IsValidLanguage(langCode string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	target := strings.ToLower(strings.TrimSpace(langCode))
	for _, code := range c.languageCodes {
		if code == target {
			return true
		}
	}
	return false
}

// GetTaxonomyConfig returns the raw parsed taxonomy configuration.
func (c *Cache) GetTaxonomyConfig() *apiclient.TaxonomyConfig {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.taxonomy
}

// GetLanguagesConfig returns the raw parsed languages configuration.
func (c *Cache) GetLanguagesConfig() *apiclient.LanguagesConfig {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.languages
}
