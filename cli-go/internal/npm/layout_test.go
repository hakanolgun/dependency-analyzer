package npm

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetectInstallLayout(t *testing.T) {
	root := t.TempDir()
	if g := DetectInstallLayout(root); g != LayoutMissingNodeModules {
		t.Fatalf("empty project: got %v", g)
	}
	nm := filepath.Join(root, "node_modules")
	if err := os.MkdirAll(nm, 0o755); err != nil {
		t.Fatal(err)
	}
	if g := DetectInstallLayout(root); g != LayoutMissingNodeModules {
		t.Fatalf("empty node_modules: got %v", g)
	}
	if err := os.WriteFile(filepath.Join(nm, ".keep"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if g := DetectInstallLayout(root); g != LayoutNodeModules {
		t.Fatalf("non-empty node_modules: got %v", g)
	}
	if err := os.WriteFile(filepath.Join(root, ".pnp.cjs"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	_ = os.RemoveAll(nm)
	if g := DetectInstallLayout(root); g != LayoutYarnPnP {
		t.Fatalf("pnp without nm: got %v", g)
	}
}
