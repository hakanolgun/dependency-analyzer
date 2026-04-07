package npm

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/hakanolgun/dependency-analyzer/cli-go/internal/engine"
)

var githubRepoPathRe = regexp.MustCompile(`github\.com[/:]([^/]+)/([^/.]+)`)

const githubLanguagesMinInterval = 350 * time.Millisecond

var (
	githubLanguagesThrottleMu sync.Mutex
	githubLanguagesLastAt     time.Time
)

func throttleBeforeGitHubLanguagesFetch() {
	githubLanguagesThrottleMu.Lock()
	defer githubLanguagesThrottleMu.Unlock()
	if !githubLanguagesLastAt.IsZero() {
		elapsed := time.Since(githubLanguagesLastAt)
		if elapsed < githubLanguagesMinInterval {
			time.Sleep(githubLanguagesMinInterval - elapsed)
		}
	}
	githubLanguagesLastAt = time.Now()
}

func parseGitHubRetryAfter(h http.Header) time.Duration {
	ra := strings.TrimSpace(h.Get("Retry-After"))
	if ra == "" {
		return 2 * time.Second
	}
	if secs, err := strconv.ParseInt(ra, 10, 64); err == nil && secs > 0 {
		if secs > 120 {
			return 120 * time.Second
		}
		return time.Duration(secs) * time.Second
	}
	return 2 * time.Second
}

func primaryLanguageFromLanguagesBody(body []byte) (string, error) {
	var m map[string]int64
	if err := json.Unmarshal(body, &m); err == nil && len(m) > 0 {
		return languageWithMaxBytes(m), nil
	}
	var mf map[string]float64
	if err := json.Unmarshal(body, &mf); err != nil {
		return "", fmt.Errorf("languages json: %w", err)
	}
	best := ""
	var maxV float64
	for lang, n := range mf {
		if n > maxV {
			maxV = n
			best = lang
		}
	}
	return best, nil
}

func languageWithMaxBytes(m map[string]int64) string {
	best := ""
	var maxB int64
	for lang, n := range m {
		if n > maxB {
			maxB = n
			best = lang
		}
	}
	return best
}

func parseGitHubOwnerRepo(repositoryJSON json.RawMessage) string {
	if len(repositoryJSON) == 0 {
		return ""
	}
	var s string
	if json.Unmarshal(repositoryJSON, &s) == nil && s != "" {
		return extractOwnerRepoFromURL(s)
	}
	var obj struct {
		URL string `json:"url"`
	}
	if json.Unmarshal(repositoryJSON, &obj) == nil && obj.URL != "" {
		return extractOwnerRepoFromURL(obj.URL)
	}
	return ""
}

func extractOwnerRepoFromURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if strings.HasPrefix(strings.ToLower(raw), "github:") {
		tail := strings.TrimSpace(raw[7:])
		tail = strings.TrimSuffix(tail, ".git")
		parts := strings.SplitN(tail, "/", 2)
		if len(parts) == 2 && parts[0] != "" && parts[1] != "" {
			return parts[0] + "/" + parts[1]
		}
	}
	raw = strings.TrimPrefix(raw, "git+")
	raw = strings.TrimSuffix(raw, ".git")
	raw = strings.ReplaceAll(raw, "git://", "https://")
	m := githubRepoPathRe.FindStringSubmatch(raw)
	if len(m) != 3 {
		return ""
	}
	return m[1] + "/" + m[2]
}

func hasNpmBinField(binJSON json.RawMessage) bool {
	if len(binJSON) == 0 {
		return false
	}
	var s string
	if json.Unmarshal(binJSON, &s) == nil {
		return strings.TrimSpace(s) != ""
	}
	var m map[string]any
	if json.Unmarshal(binJSON, &m) == nil {
		return len(m) > 0
	}
	return false
}

// strongNativeGitHubLanguage is true when GitHub's primary repo language points to a
// non-trivial native / compiled implementation (heuristic).
func strongNativeGitHubLanguage(lang string) bool {
	switch strings.TrimSpace(lang) {
	case "Rust", "Go", "C", "C++", "Objective-C", "Swift", "Kotlin", "Java", "Zig", "C#", "Assembly":
		return true
	default:
		return false
	}
}

// binaryShimNativePresence returns 1 when the on-disk package looks like a thin JS wrapper
// around large prebuilt assets (bytes ≫ source lines). hasBin limits false positives from
// large documentation-only packages.
func binaryShimNativePresence(pkgBytes int64, sloc int, hasBin bool) float64 {
	if !hasBin || pkgBytes < 120_000 {
		return 0
	}
	if sloc < 1 {
		return 1
	}
	if pkgBytes/int64(sloc) >= 2000 {
		return 1
	}
	return 0
}

// FetchGitHubPrimaryLanguage calls GET /repos/{owner}/{repo}/languages and returns the
// language with the largest byte count (same notion as GitHub's repo "language" summary).
// It throttles consecutive requests (350ms minimum gap) and retries once on 403/429 after
// Retry-After (capped at 120s).
func FetchGitHubPrimaryLanguage(ctx context.Context, client *http.Client, ownerRepo string) (string, error) {
	if client == nil || ownerRepo == "" {
		return "", fmt.Errorf("missing client or ownerRepo")
	}
	parts := strings.Split(ownerRepo, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", fmt.Errorf("invalid ownerRepo")
	}
	u := fmt.Sprintf(
		"https://api.github.com/repos/%s/%s/languages",
		url.PathEscape(parts[0]),
		url.PathEscape(parts[1]),
	)

	var lastErr error
	for attempt := 0; attempt < 2; attempt++ {
		if attempt == 0 {
			throttleBeforeGitHubLanguagesFetch()
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
		if err != nil {
			return "", err
		}
		req.Header.Set("Accept", "application/vnd.github+json")
		req.Header.Set("User-Agent", "dependency-analyzer-native-presence")

		resp, err := client.Do(req)
		if err != nil {
			return "", err
		}
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
		_ = resp.Body.Close()

		switch resp.StatusCode {
		case http.StatusOK:
			lang, err := primaryLanguageFromLanguagesBody(body)
			if err != nil {
				return "", err
			}
			if lang == "" {
				return "", nil
			}
			return lang, nil
		case http.StatusForbidden, http.StatusTooManyRequests:
			lastErr = fmt.Errorf("github api: %s: %s", resp.Status, strings.TrimSpace(string(body)))
			if attempt == 0 {
				select {
				case <-ctx.Done():
					return "", ctx.Err()
				case <-time.After(parseGitHubRetryAfter(resp.Header)):
				}
				continue
			}
			return "", lastErr
		default:
			return "", fmt.Errorf("github api: %s: %s", resp.Status, strings.TrimSpace(string(body)))
		}
	}
	if lastErr != nil {
		return "", lastErr
	}
	return "", nil
}

func finalizeNativeFromBinaryShim(base float64, pkgBytes int64, sloc int, hasBin bool) float64 {
	return clamp(max(base, binaryShimNativePresence(pkgBytes, sloc, hasBin)))
}

func applyGitHubStrongNativeLanguage(m *engine.Metrics, lang string) {
	if strongNativeGitHubLanguage(lang) {
		m.Native = clamp(max(m.Native, 1.0))
	}
}

func httpClientForNativeHints(preferred *http.Client) *http.Client {
	if preferred != nil {
		return preferred
	}
	return &http.Client{Timeout: 8 * time.Second}
}
