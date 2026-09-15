// Package catalog holds the shared shape of the OSC service catalog as published in
// this repository's catalog/services.json mirror. The mirror is regenerated on a
// schedule from the full OSC catalog, which needs an API key that end users do not
// have. The provider uses it to validate configurations for services the workspace
// has not subscribed to yet, and the guide pages on the Terraform Registry are
// generated from the same data.
package catalog

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"
)

// DefaultMirrorURL is where the provider reads the mirror from unless the
// OSC_CATALOG_MIRROR_URL environment variable overrides it.
const DefaultMirrorURL = "https://raw.githubusercontent.com/EyevinnOSC/terraform-provider-osc/main/catalog/services.json"

// MirrorURLEnv overrides DefaultMirrorURL, mainly for tests and forks.
const MirrorURLEnv = "OSC_CATALOG_MIRROR_URL"

// GuidesBaseURL is where the generated per-service guide pages are published.
const GuidesBaseURL = "https://registry.terraform.io/providers/EyevinnOSC/osc/latest/docs/guides/"

// Option is one entry of a service's instance option schema.
type Option struct {
	Name                string   `json:"name"`
	Label               string   `json:"label,omitempty"`
	Description         string   `json:"description,omitempty"`
	ExtendedDescription string   `json:"extendedDescription,omitempty"`
	Type                string   `json:"type"`
	Enum                []string `json:"enums,omitempty"`
	Mandatory           bool     `json:"mandatory"`
	Default             string   `json:"default,omitempty"`
	RegexValidator      string   `json:"regexValidator,omitempty"`
	Sensitive           bool     `json:"sensitive,omitempty"`
}

// Service is a published catalog service.
type Service struct {
	ServiceID        string   `json:"serviceId"`
	Title            string   `json:"title"`
	Description      string   `json:"description,omitempty"`
	Category         string   `json:"category,omitempty"`
	DocumentationURL string   `json:"documentationUrl,omitempty"`
	RepoURL          string   `json:"repoUrl,omitempty"`
	ServiceType      string   `json:"serviceType"`
	Status           string   `json:"status"`
	APIURL           string   `json:"apiUrl"`
	Options          []Option `json:"options"`
}

// Mirror is the content of catalog/services.json.
type Mirror struct {
	GeneratedAt time.Time `json:"generatedAt"`
	Environment string    `json:"environment"`
	Services    []Service `json:"services"`
}

// MirrorURL returns the configured mirror URL.
func MirrorURL() string {
	if u := os.Getenv(MirrorURLEnv); u != "" {
		return u
	}
	return DefaultMirrorURL
}

// Fetch downloads and decodes the mirror.
func Fetch(ctx context.Context, url string) (*Mirror, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d fetching catalog mirror %s", resp.StatusCode, url)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 16<<20))
	if err != nil {
		return nil, err
	}
	var m Mirror
	if err := json.Unmarshal(body, &m); err != nil {
		return nil, fmt.Errorf("decoding catalog mirror: %w", err)
	}
	return &m, nil
}

// Find returns the service with the given id, or nil.
func (m *Mirror) Find(serviceID string) *Service {
	if m == nil {
		return nil
	}
	for i := range m.Services {
		if m.Services[i].ServiceID == serviceID {
			return &m.Services[i]
		}
	}
	return nil
}

// Suggest returns up to n service ids that resemble the given id, closest first.
// It matches on edit distance and on shared name fragments, so "eyevinn-couchdb"
// still finds "apache-couchdb".
func (m *Mirror) Suggest(serviceID string, n int) []string {
	if m == nil {
		return nil
	}
	type scored struct {
		id    string
		score int
	}
	needle := strings.ToLower(serviceID)
	frags := strings.FieldsFunc(needle, func(r rune) bool { return r == '-' || r == '_' || r == '.' })
	var out []scored
	for _, s := range m.Services {
		id := strings.ToLower(s.ServiceID)
		contributor, rest, _ := strings.Cut(id, "-")
		score := Levenshtein(needle, id)
		for _, f := range frags {
			switch {
			case len(f) >= 3 && strings.Contains(rest, f):
				score -= len(f) // shared fragment in the service name is a strong signal
			case f == contributor:
				score-- // a shared contributor prefix says little on its own
			}
		}
		if strings.Contains(id, needle) || strings.Contains(needle, id) {
			score -= 3
		}
		out = append(out, scored{s.ServiceID, score})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].score != out[j].score {
			return out[i].score < out[j].score
		}
		return out[i].id < out[j].id
	})
	limit := len(needle)/2 + 2
	var ids []string
	for _, c := range out {
		if len(ids) >= n || c.score > limit {
			break
		}
		// Keep only candidates close to the best match, so a clear winner is not
		// padded with weak ones that merely share the contributor prefix.
		if len(ids) > 0 && c.score > out[0].score+2 {
			break
		}
		ids = append(ids, c.id)
	}
	return ids
}

// GuideURL returns the registry URL of the generated guide page for a service.
func GuideURL(serviceID string) string {
	return GuidesBaseURL + GuideSlug(serviceID)
}

// GuideSlug is the file name (without extension) of a service's guide page.
func GuideSlug(serviceID string) string {
	return strings.ToLower(serviceID)
}

// Levenshtein returns the edit distance between two strings.
func Levenshtein(a, b string) int {
	ra, rb := []rune(a), []rune(b)
	prev := make([]int, len(rb)+1)
	curr := make([]int, len(rb)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(ra); i++ {
		curr[0] = i
		for j := 1; j <= len(rb); j++ {
			cost := 1
			if ra[i-1] == rb[j-1] {
				cost = 0
			}
			curr[j] = min(prev[j]+1, curr[j-1]+1, prev[j-1]+cost)
		}
		prev, curr = curr, prev
	}
	return prev[len(rb)]
}
