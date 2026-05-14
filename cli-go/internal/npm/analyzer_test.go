package npm

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAnalyzeProjectBasic(t *testing.T) {
	root := t.TempDir()

	rootPkg := `{
  "name": "fixture",
  "dependencies": {
    "pkg-a": "^1.0.0"
  }
}`
	if err := os.WriteFile(filepath.Join(root, "package.json"), []byte(rootPkg), 0o644); err != nil {
		t.Fatal(err)
	}

	depDir := filepath.Join(root, "node_modules", "pkg-a")
	if err := os.MkdirAll(depDir, 0o755); err != nil {
		t.Fatal(err)
	}

	depPkg := `{
  "name": "pkg-a",
  "version": "1.0.0",
  "dependencies": {
    "dep-x": "1.0.0"
  },
  "peerDependencies": {
    "react": "^19.0.0"
  }
}`
	if err := os.WriteFile(filepath.Join(depDir, "package.json"), []byte(depPkg), 0o644); err != nil {
		t.Fatal(err)
	}

	src := `export class A {
  public run() {
    if (true) { return 1; }
    return 0;
  }
}
`
	if err := os.WriteFile(filepath.Join(depDir, "index.ts"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}

	report, err := AnalyzeProjectWithRegistry(root, nil)
	if err != nil {
		t.Fatalf("AnalyzeProject failed: %v", err)
	}

	if report.ScannedCount != 1 {
		t.Fatalf("ScannedCount=%d, want 1", report.ScannedCount)
	}
	if len(report.Dependencies) != 1 {
		t.Fatalf("deps=%d, want 1", len(report.Dependencies))
	}
	got := report.Dependencies[0]
	if got.Error != "" {
		t.Fatalf("unexpected error: %s", got.Error)
	}
	if got.Score < 0 || got.Score > 100 {
		t.Fatalf("score out of range: %d", got.Score)
	}
}

func TestDefaultSkipsDevDependencies(t *testing.T) {
	root := t.TempDir()
	rootPkg := `{
  "name": "fixture",
  "dependencies": { "a": "1.0.0" },
  "devDependencies": { "b": "2.0.0" }
}`
	if err := os.WriteFile(filepath.Join(root, "package.json"), []byte(rootPkg), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"a", "b"} {
		depDir := filepath.Join(root, "node_modules", name)
		if err := os.MkdirAll(depDir, 0o755); err != nil {
			t.Fatal(err)
		}
		mini := `{"name": "` + name + `", "version": "1.0.0", "dependencies": {}, "peerDependencies": {}}`
		if err := os.WriteFile(filepath.Join(depDir, "package.json"), []byte(mini), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(depDir, "index.js"), []byte("export const x = 1;\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	// Default behavior: only dependencies are scanned, devDependencies are excluded.
	report, err := AnalyzeProjectWithRegistry(root, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Dependencies) != 1 {
		t.Fatalf("want 1 dep (only prod), got %d", len(report.Dependencies))
	}
	if report.Dependencies[0].Name != "a" {
		t.Fatalf("expected dep 'a', got %q", report.Dependencies[0].Name)
	}
}

func TestIncludeDevDependencies(t *testing.T) {
	root := t.TempDir()
	rootPkg := `{
  "name": "fixture",
  "dependencies": { "a": "1.0.0" },
  "devDependencies": { "b": "2.0.0", "a": "1.1.0" }
}`
	if err := os.WriteFile(filepath.Join(root, "package.json"), []byte(rootPkg), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"a", "b"} {
		depDir := filepath.Join(root, "node_modules", name)
		if err := os.MkdirAll(depDir, 0o755); err != nil {
			t.Fatal(err)
		}
		mini := `{"name": "` + name + `", "version": "1.0.0", "dependencies": {}, "peerDependencies": {}}`
		if err := os.WriteFile(filepath.Join(depDir, "package.json"), []byte(mini), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(depDir, "index.js"), []byte("export const x = 1;\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	// With --dev flag: both dependencies and devDependencies are scanned.
	opts := AnalyzeOptions{AllowGhost: true, IncludeDevDependencies: true}
	report, err := AnalyzeProjectWithRegistryOpts(root, nil, opts)
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Dependencies) != 2 {
		t.Fatalf("want 2 merged deps, got %d", len(report.Dependencies))
	}
	// Alphabetical order: a before b; devDependencies overwrote a's version.
	var foundA, foundB bool
	for _, d := range report.Dependencies {
		if d.Name == "a" && d.Version == "1.1.0" {
			foundA = true
		}
		if d.Name == "b" {
			foundB = true
		}
	}
	if !foundA || !foundB {
		t.Fatalf("merge overwrite or keys: %+v", report.Dependencies)
	}
}

func TestAnalyzeProjectSymlinkedNodeModules(t *testing.T) {
	root := t.TempDir()
	store := filepath.Join(root, ".pnpm-store", "pkg-a")
	if err := os.MkdirAll(store, 0o755); err != nil {
		t.Fatal(err)
	}
	rootPkg := `{
  "name": "fixture",
  "dependencies": { "pkg-a": "^1.0.0" }
}`
	if err := os.WriteFile(filepath.Join(root, "package.json"), []byte(rootPkg), 0o644); err != nil {
		t.Fatal(err)
	}
	nm := filepath.Join(root, "node_modules")
	if err := os.MkdirAll(nm, 0o755); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(nm, "pkg-a")
	if err := os.Symlink(store, link); err != nil {
		t.Fatal(err)
	}
	depPkg := `{"name":"pkg-a","version":"1.0.0","dependencies":{},"peerDependencies":{}}`
	if err := os.WriteFile(filepath.Join(store, "package.json"), []byte(depPkg), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(store, "index.js"), []byte("export const x = 1;\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	report, err := AnalyzeProjectWithRegistry(root, nil)
	if err != nil {
		t.Fatal(err)
	}
	if report.ScannedCount != 1 {
		t.Fatalf("ScannedCount=%d", report.ScannedCount)
	}
	if report.Dependencies[0].Error != "" {
		t.Fatal(report.Dependencies[0].Error)
	}
}

func TestAnalyzeProjectNoGhostMissingNodeModules(t *testing.T) {
	root := t.TempDir()
	rootPkg := `{"name":"x","dependencies":{"missing-pkg":"1.0.0"}}`
	if err := os.WriteFile(filepath.Join(root, "package.json"), []byte(rootPkg), 0o644); err != nil {
		t.Fatal(err)
	}
	opts := AnalyzeOptions{AllowGhost: false}
	report, err := AnalyzeProjectWithRegistryOpts(root, nil, opts)
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Dependencies) != 1 {
		t.Fatalf("deps %d", len(report.Dependencies))
	}
	if report.Dependencies[0].Error == "" {
		t.Fatal("expected error when node_modules missing and --no-ghost")
	}
}
