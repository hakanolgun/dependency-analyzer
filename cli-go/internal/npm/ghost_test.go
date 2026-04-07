package npm

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestGhostPathEndToEnd(t *testing.T) {
	tgz := mustMiniNpmTGZ(t, "ghost-pkg", "1.0.0", "export const x = 1;\n")

	var base string
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/ghost-pkg/1.0.0":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"dist": map[string]string{"tarball": base + "/pkg.tgz"},
			})
		case "/pkg.tgz":
			w.Header().Set("Content-Type", "application/octet-stream")
			_, _ = w.Write(tgz)
		default:
			http.NotFound(w, r)
		}
	})
	srv := httptest.NewServer(h)
	defer srv.Close()
	base = srv.URL

	c := &Client{
		HTTP:            srv.Client(),
		RegistryBaseURL: base,
	}
	ctx := context.Background()
	proj := t.TempDir()
	res, tmp := analyzeDependencyGhost(ctx, c, proj, "ghost-pkg", "1.0.0")
	if tmp != "" {
		defer func() { _ = os.RemoveAll(tmp) }()
	}
	if res.Error != "" {
		t.Fatal(res.Error)
	}
	if res.Score < 0 || res.Score > 100 {
		t.Fatalf("score %d", res.Score)
	}
	if res.Version != "1.0.0" {
		t.Fatalf("version %q", res.Version)
	}
}

func TestFetchTarballURL(t *testing.T) {
	tgz := mustMiniNpmTGZ(t, "x", "1.0.0", "")
	var base string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/x/1.0.0":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"dist": map[string]string{"tarball": base + "/ball.tgz"},
			})
		case "/ball.tgz":
			_, _ = w.Write(tgz)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	base = srv.URL

	c := &Client{HTTP: srv.Client(), RegistryBaseURL: base}
	u, err := c.FetchTarballURL(context.Background(), "x", "1.0.0")
	if err != nil {
		t.Fatal(err)
	}
	if u != base+"/ball.tgz" {
		t.Fatalf("got %s", u)
	}
}

func mustMiniNpmTGZ(t *testing.T, name, version, js string) []byte {
	t.Helper()
	pkgJSON := `{"name":"` + name + `","version":"` + version + `","dependencies":{},"peerDependencies":{}}`
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)
	add := func(name, body string) {
		h := &tar.Header{
			Name:     name,
			Mode:     0o644,
			Size:     int64(len(body)),
			Typeflag: tar.TypeReg,
		}
		if err := tw.WriteHeader(h); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write([]byte(body)); err != nil {
			t.Fatal(err)
		}
	}
	add("package/package.json", pkgJSON)
	add("package/index.js", js)
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}
