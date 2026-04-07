package npm

import (
	"context"
	"encoding/json"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/hakanolgun/dependency-analyzer/cli-go/internal/engine"
)

func TestParseGitHubOwnerRepo(t *testing.T) {
	raw := json.RawMessage(`"git+https://github.com/biomejs/biome.git"`)
	if got := parseGitHubOwnerRepo(raw); got != "biomejs/biome" {
		t.Fatalf("got %q", got)
	}
	obj := json.RawMessage(`{"type":"git","url":"git+https://github.com/foo/bar.git"}`)
	if got := parseGitHubOwnerRepo(obj); got != "foo/bar" {
		t.Fatalf("got %q", got)
	}
}

func TestStrongNativeGitHubLanguage(t *testing.T) {
	if !strongNativeGitHubLanguage("Rust") {
		t.Fatal("Rust should match")
	}
	if strongNativeGitHubLanguage("TypeScript") {
		t.Fatal("TypeScript should not match")
	}
}

func TestPrimaryLanguageFromLanguagesBody(t *testing.T) {
	body := []byte(`{"Rust":21494797,"JavaScript":1884117,"TypeScript":871838}`)
	lang, err := primaryLanguageFromLanguagesBody(body)
	if err != nil {
		t.Fatal(err)
	}
	if lang != "Rust" {
		t.Fatalf("want Rust, got %q", lang)
	}
}

func TestFinalizeNativeFromBinaryShim_BiomeLike(t *testing.T) {
	// @biomejs/biome: ~95 lines of JS in bin, ~700KiB on disk, declares bin.
	const pkgBytes int64 = 704_768
	const sloc = 0
	const hasBin = true
	base := 0.0
	got := finalizeNativeFromBinaryShim(base, pkgBytes, sloc, hasBin)
	if got < 1 {
		t.Fatalf("expected native 1, got %v", got)
	}
	m := engine.Metrics{Native: got, Volume: 0, APISurface: 0, Entanglement: 0, LogicComplexity: 0}
	score := engine.ToPercentageScore(engine.ComputeNormalized(m))
	if score < 40 {
		t.Fatalf("biome-like metrics should score >= 40, got %d", score)
	}
}

func TestBiomejsBiomeReplaceabilityScore_realPack(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: npm pack + optional GitHub")
	}
	if _, err := exec.LookPath("npm"); err != nil {
		t.Skip("npm not available")
	}
	if _, err := exec.LookPath("tar"); err != nil {
		t.Skip("tar not available")
	}

	tmp := t.TempDir()
	cmd := exec.Command("npm", "pack", "@biomejs/biome@2.4.10", "--pack-destination", tmp)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("npm pack: %v\n%s", err, out)
	}
	matches, err := filepath.Glob(filepath.Join(tmp, "*.tgz"))
	if err != nil || len(matches) != 1 {
		t.Fatalf("expected one .tgz in %s, got %v err=%v", tmp, matches, err)
	}
	if err := exec.Command("tar", "-xzf", matches[0], "-C", tmp).Run(); err != nil {
		t.Fatal(err)
	}
	extDir := filepath.Join(tmp, "package")

	metrics, ghRepo, err := collectMetrics(extDir)
	if err != nil {
		t.Fatal(err)
	}
	if ghRepo != "biomejs/biome" {
		t.Fatalf("github repo parse: got %q", ghRepo)
	}

	ctx := context.Background()
	applyGitHubNativeLanguageHint(ctx, NewClient(), ghRepo, &metrics)

	norm := engine.ComputeNormalized(metrics)
	score := engine.ToPercentageScore(norm)
	if score < 40 {
		t.Fatalf("expected @biomejs/biome score >= 40, got %d (norm=%.3f metrics=%+v)", score, norm, metrics)
	}
}
