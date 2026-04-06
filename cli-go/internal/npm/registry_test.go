package npm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestFormatTimeSince(t *testing.T) {
	now := time.Date(2026, 4, 6, 12, 0, 0, 0, time.UTC)
	cases := []struct {
		days int
		want string
	}{
		{0, "Today"},
		{1, "1 day ago"},
		{5, "5 days ago"},
		{35, "1 month ago"},
		{400, "1 year ago"},
	}
	for _, tc := range cases {
		past := now.AddDate(0, 0, -tc.days)
		got := formatTimeSince(past, now)
		if got != tc.want {
			t.Fatalf("days=%d: got %q want %q", tc.days, got, tc.want)
		}
	}
}

func TestClient_FetchRegistryMeta_mocked(t *testing.T) {
	regDoc := map[string]any{
		"dist-tags": map[string]string{"latest": "2.0.0"},
		"time": map[string]string{
			"2.0.0":    "2025-01-15T10:00:00.000Z",
			"modified": "2025-01-15T10:00:00.000Z",
		},
		"versions": map[string]any{
			"2.0.0": map[string]string{},
		},
		"repository": map[string]string{"url": "git+https://github.com/foo/bar.git"},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/reg/"):
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(regDoc)
		case strings.HasPrefix(r.URL.Path, "/dl/point/last-week/"):
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]int{"downloads": 42_000})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	c := NewClient()
	c.RegistryBaseURL = srv.URL + "/reg"
	c.DownloadsBaseURL = srv.URL + "/dl/point/last-week"
	c.HTTP = srv.Client()

	meta, err := c.FetchRegistryMeta(context.Background(), "left-pad", false)
	if err != nil {
		t.Fatal(err)
	}
	if meta.LatestVersion != "2.0.0" {
		t.Fatalf("LatestVersion=%q", meta.LatestVersion)
	}
	if !strings.Contains(meta.RepoURL, "github.com/foo/bar") {
		t.Fatalf("RepoURL=%q", meta.RepoURL)
	}
	if meta.WeeklyDownloads == nil || *meta.WeeklyDownloads != 42_000 {
		t.Fatalf("downloads=%v", meta.WeeklyDownloads)
	}
	if meta.IsMaintained != "yes" {
		t.Fatalf("maintained=%q", meta.IsMaintained)
	}
}
