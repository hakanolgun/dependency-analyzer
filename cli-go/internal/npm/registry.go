package npm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

const (
	registryBaseURL = "https://registry.npmjs.org"
	downloadsAPI    = "https://api.npmjs.org/downloads/point/last-week"
	rnDirectoryAPI  = "https://reactnative.directory/api/libraries"
)

// Client fetches npm registry metadata and optional React Native directory data.
type Client struct {
	HTTP *http.Client

	// Optional overrides for tests (must include scheme, no trailing slash path for registry root).
	RegistryBaseURL  string
	DownloadsBaseURL string
	RNDirectoryURL   string

	mu             sync.Mutex
	lastDownloadAt time.Time
}

func NewClient() *Client {
	return &Client{
		HTTP: &http.Client{Timeout: 30 * time.Second},
	}
}

// RegistryMeta mirrors fields needed for DependencyResult from npm APIs.
type RegistryMeta struct {
	LatestVersion       string
	RepoURL             string
	LastUpdateDate      string
	TimeSinceLastUpdate string
	IsMaintained        string
	WeeklyDownloads     *int

	IsReactNativeLib bool
	NewArchitecture  *bool
}

type npmRegistryDoc struct {
	DistTags struct {
		Latest string `json:"latest"`
	} `json:"dist-tags"`
	Time     map[string]string `json:"time"`
	Versions map[string]struct {
		Deprecated string `json:"deprecated"`
	} `json:"versions"`
	Repository *struct {
		URL string `json:"url"`
	} `json:"repository"`
	Homepage string `json:"homepage"`
}

type downloadsPoint struct {
	Downloads int `json:"downloads"`
}

type rnSearchResponse struct {
	Libraries []rnLibrary `json:"libraries"`
}

type rnLibrary struct {
	NpmPkg          string `json:"npmPkg"`
	NewArchitecture bool   `json:"newArchitecture"`
	Unmaintained    bool   `json:"unmaintained"`
	Github          *struct {
		NewArchitecture bool   `json:"newArchitecture"`
		HasNativeCode   bool   `json:"hasNativeCode"`
		ModuleType      string `json:"moduleType"`
	} `json:"github"`
}

// FetchRegistryMeta loads registry + downloads (+ optional RN) for one package.
func (c *Client) FetchRegistryMeta(ctx context.Context, name string, hasReactNativeProject bool) (RegistryMeta, error) {
	var out RegistryMeta

	regRoot := registryBaseURL
	if c.RegistryBaseURL != "" {
		regRoot = strings.TrimSuffix(c.RegistryBaseURL, "/")
	}
	regURL := fmt.Sprintf("%s/%s", regRoot, url.PathEscape(name))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, regURL, nil)
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
		return out, fmt.Errorf("registry %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}

	var doc npmRegistryDoc
	if err := json.NewDecoder(resp.Body).Decode(&doc); err != nil {
		return out, err
	}

	latest := doc.DistTags.Latest
	out.LatestVersion = latest

	if latest != "" {
		if dep, ok := doc.Versions[latest]; ok && dep.Deprecated != "" {
			out.IsMaintained = "no"
		} else if t, ok := doc.Time[latest]; ok && t != "" {
			out.LastUpdateDate = t
			if ts, err := time.Parse(time.RFC3339, t); err == nil {
				out.TimeSinceLastUpdate = formatTimeSince(ts, time.Now())
				twoYears := 2 * 365 * 24 * time.Hour
				if time.Since(ts) > twoYears && out.IsMaintained == "" {
					out.IsMaintained = "unlikely"
				}
			}
		} else if doc.Time["modified"] != "" {
			out.LastUpdateDate = doc.Time["modified"]
			if ts, err := time.Parse(time.RFC3339, doc.Time["modified"]); err == nil {
				out.TimeSinceLastUpdate = formatTimeSince(ts, time.Now())
			}
		}
		if out.IsMaintained == "" {
			out.IsMaintained = "yes"
		}
	}

	out.RepoURL = repoURLFromDoc(&doc, name)

	dl, err := c.fetchDownloadsThrottled(ctx, name)
	if err == nil {
		out.WeeklyDownloads = &dl
	}

	if hasReactNativeProject {
		rn, err := c.fetchRNLibrary(ctx, name)
		if err == nil {
			out.IsReactNativeLib = rn.IsReactNativeLib
			out.NewArchitecture = rn.NewArchitecture
			if rn.UnmaintainedOverride {
				out.IsMaintained = "no"
			}
		}
	}

	return out, nil
}

func repoURLFromDoc(doc *npmRegistryDoc, name string) string {
	var raw string
	if doc.Repository != nil && doc.Repository.URL != "" {
		raw = doc.Repository.URL
	} else if doc.Homepage != "" {
		raw = doc.Homepage
	} else {
		return fmt.Sprintf("https://www.npmjs.com/package/%s", url.PathEscape(name))
	}
	raw = strings.TrimPrefix(raw, "git+")
	raw = strings.TrimSuffix(raw, ".git")
	raw = strings.ReplaceAll(raw, "git://", "https://")
	return raw
}

func (c *Client) fetchDownloadsThrottled(ctx context.Context, name string) (int, error) {
	c.mu.Lock()
	if !c.lastDownloadAt.IsZero() {
		elapsed := time.Since(c.lastDownloadAt)
		if elapsed < 350*time.Millisecond {
			time.Sleep(350*time.Millisecond - elapsed)
		}
	}
	c.lastDownloadAt = time.Now()
	c.mu.Unlock()

	dlRoot := downloadsAPI
	if c.DownloadsBaseURL != "" {
		dlRoot = strings.TrimSuffix(c.DownloadsBaseURL, "/")
	}
	u := fmt.Sprintf("%s/%s", dlRoot, url.PathEscape(name))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return 0, err
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("downloads API: %s", resp.Status)
	}
	var pt downloadsPoint
	if err := json.NewDecoder(resp.Body).Decode(&pt); err != nil {
		return 0, err
	}
	return pt.Downloads, nil
}

type rnOutcome struct {
	IsReactNativeLib     bool
	NewArchitecture      *bool
	UnmaintainedOverride bool
}

func (c *Client) fetchRNLibrary(ctx context.Context, name string) (rnOutcome, error) {
	var out rnOutcome
	rnRoot := rnDirectoryAPI
	if c.RNDirectoryURL != "" {
		rnRoot = strings.TrimSuffix(c.RNDirectoryURL, "/")
	}
	u := rnRoot + "?search=" + url.QueryEscape(name)
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
		return out, errors.New("reactnative.directory error")
	}
	var body rnSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return out, err
	}
	var match *rnLibrary
	for i := range body.Libraries {
		if body.Libraries[i].NpmPkg == name {
			match = &body.Libraries[i]
			break
		}
	}
	if match == nil {
		// TS default: assume New Arch OK when not listed (pure JS).
		t := true
		out.NewArchitecture = &t
		return out, nil
	}

	out.IsReactNativeLib = true
	na := match.NewArchitecture ||
		(match.Github != nil && (match.Github.NewArchitecture ||
			!match.Github.HasNativeCode ||
			match.Github.ModuleType == "expo"))
	t := na
	out.NewArchitecture = &t
	if match.Unmaintained {
		out.UnmaintainedOverride = true
	}
	return out, nil
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
