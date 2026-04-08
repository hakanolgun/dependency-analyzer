package gomod

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
	"unicode"
	"github.com/hakanolgun/dependency-analyzer/cli-go/internal/engine"
)

const (
	defaultProxyURL = "https://proxy.golang.org"
)

// Client fetches Go module metadata and source from the Go module proxy.
type Client struct {
	HTTP     *http.Client
	ProxyURL string // override for tests

	mu     sync.Mutex
	lastAt time.Time
}

// NewClient creates a default Go proxy client with a 30s timeout.
func NewClient() *Client {
	return &Client{
		HTTP: &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *Client) proxyRoot() string {
	if c.ProxyURL != "" {
		return strings.TrimSuffix(c.ProxyURL, "/")
	}
	return defaultProxyURL
}

// throttle ensures at least 200ms between requests to the proxy.
func (c *Client) throttle() {
	c.mu.Lock()
	if !c.lastAt.IsZero() {
		elapsed := time.Since(c.lastAt)
		if elapsed < 200*time.Millisecond {
			time.Sleep(200*time.Millisecond - elapsed)
		}
	}
	c.lastAt = time.Now()
	c.mu.Unlock()
}

// proxyLatestResponse mirrors the JSON payload from proxy.golang.org/{module}/@latest.
type proxyLatestResponse struct {
	Version string `json:"Version"`
	Time    string `json:"Time"`
	Origin  *struct {
		URL string `json:"URL"`
	} `json:"Origin"`
}

// ModuleMeta holds metadata fetched from the Go proxy.
type ModuleMeta struct {
	LatestVersion       string
	LastUpdateDate      string
	TimeSinceLastUpdate string
	IsMaintained        string // yes | unlikely
	RepoURL             string
}

// FetchModuleMeta fetches latest version info from proxy.golang.org.
func (c *Client) FetchModuleMeta(ctx context.Context, modulePath string) (ModuleMeta, error) {
	var out ModuleMeta
	c.throttle()

	encoded := encodeModulePath(modulePath)
	u := fmt.Sprintf("%s/%s/@latest", c.proxyRoot(), encoded)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return out, err
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return out, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return out, fmt.Errorf("proxy %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}

	var data proxyLatestResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return out, err
	}

	out.LatestVersion = data.Version

	if data.Time != "" {
		out.LastUpdateDate = data.Time
		if ts, err := time.Parse(time.RFC3339Nano, data.Time); err == nil {
			out.TimeSinceLastUpdate = formatTimeSince(ts, time.Now())
			threeYears := 3 * 365 * 24 * time.Hour
			if time.Since(ts) > threeYears {
				out.IsMaintained = "unlikely"
			} else {
				out.IsMaintained = "yes"
			}
		}
	}

	if data.Origin != nil && data.Origin.URL != "" {
		out.RepoURL = engine.NormalizeRepoURL(data.Origin.URL)
	}

	return out, nil
}

// FetchModuleZip downloads the source zip for a specific module version.
// Returns a *zip.Reader for analysis.
func (c *Client) FetchModuleZip(ctx context.Context, modulePath, version string) (*zip.Reader, error) {
	c.throttle()

	encoded := encodeModulePath(modulePath)
	u := fmt.Sprintf("%s/%s/@v/%s.zip", c.proxyRoot(), encoded, version)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return nil, fmt.Errorf("zip download %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading zip body: %w", err)
	}

	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("opening zip: %w", err)
	}

	return reader, nil
}

// FetchModuleMod downloads the go.mod of a specific module version for dependency counting.
func (c *Client) FetchModuleMod(ctx context.Context, modulePath, version string) (string, error) {
	c.throttle()

	encoded := encodeModulePath(modulePath)
	u := fmt.Sprintf("%s/%s/@v/%s.mod", c.proxyRoot(), encoded, version)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return "", err
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("mod fetch %s: %s", resp.Status, resp.Status)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// encodeModulePath encodes a Go module path for use in proxy URLs.
// Uppercase letters are escaped with '!' prefix and lowercased.
func encodeModulePath(modulePath string) string {
	var b strings.Builder
	for _, r := range modulePath {
		if unicode.IsUpper(r) {
			b.WriteByte('!')
			b.WriteRune(unicode.ToLower(r))
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func formatTimeSince(t, now time.Time) string {
	days := int(now.Sub(t).Hours() / 24)
	if days < 0 {
		days = 0
	}
	if days == 0 {
		return "Today"
	}
	if days < 30 {
		if days == 1 {
			return "1 day ago"
		}
		return fmt.Sprintf("%d days ago", days)
	}
	months := days / 30
	if months < 12 {
		if months == 1 {
			return "1 month ago"
		}
		return fmt.Sprintf("%d months ago", months)
	}
	years := days / 365
	if years == 1 {
		return "1 year ago"
	}
	return fmt.Sprintf("%d years ago", years)
}
