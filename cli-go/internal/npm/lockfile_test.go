package npm

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveLockedVersionNpmLockV2(t *testing.T) {
	root := t.TempDir()
	lock := `{
  "lockfileVersion": 3,
  "packages": {
    "": { "name": "app" },
    "node_modules/foo": { "version": "2.3.4" },
    "node_modules/@scope/bar": { "version": "1.0.0" }
  }
}`
	if err := os.WriteFile(filepath.Join(root, "package-lock.json"), []byte(lock), 0o644); err != nil {
		t.Fatal(err)
	}
	if v, ok := ResolveLockedVersion(root, "foo"); !ok || v != "2.3.4" {
		t.Fatalf("foo: got %q ok=%v", v, ok)
	}
	if v, ok := ResolveLockedVersion(root, "@scope/bar"); !ok || v != "1.0.0" {
		t.Fatalf("@scope/bar: got %q ok=%v", v, ok)
	}
}

func TestResolveLockedVersionNpmLockV1(t *testing.T) {
	root := t.TempDir()
	lock := `{
  "lockfileVersion": 1,
  "dependencies": {
    "left-pad": { "version": "1.0.0" }
  }
}`
	if err := os.WriteFile(filepath.Join(root, "package-lock.json"), []byte(lock), 0o644); err != nil {
		t.Fatal(err)
	}
	if v, ok := ResolveLockedVersion(root, "left-pad"); !ok || v != "1.0.0" {
		t.Fatalf("got %q ok=%v", v, ok)
	}
}

func TestResolveLockedVersionYarnLock(t *testing.T) {
	root := t.TempDir()
	lock := `
"foo@^1.0.0":
  version "1.2.3"
  resolved "https://registry.yarnpkg.com/foo/-/foo-1.2.3.tgz#abc"
  integrity sha512-deadbeef
`
	if err := os.WriteFile(filepath.Join(root, "yarn.lock"), []byte(lock), 0o644); err != nil {
		t.Fatal(err)
	}
	if v, ok := ResolveLockedVersion(root, "foo"); !ok || v != "1.2.3" {
		t.Fatalf("got %q ok=%v", v, ok)
	}
}

func TestResolveLockedVersionPnpmLock(t *testing.T) {
	root := t.TempDir()
	lock := `lockfileVersion: '9.0'

importers:
  .:
    dependencies:
      lodash:
        specifier: ^4.17.21
        version: 4.17.21
    devDependencies:
      debug:
        specifier: ^4.3.0
        version: 4.3.4
`
	if err := os.WriteFile(filepath.Join(root, "pnpm-lock.yaml"), []byte(lock), 0o644); err != nil {
		t.Fatal(err)
	}
	if v, ok := ResolveLockedVersion(root, "lodash"); !ok || v != "4.17.21" {
		t.Fatalf("lodash: got %q ok=%v", v, ok)
	}
	if v, ok := ResolveLockedVersion(root, "debug"); !ok || v != "4.3.4" {
		t.Fatalf("debug: got %q ok=%v", v, ok)
	}
}

func TestPackageJSONVersionUsableForGhost(t *testing.T) {
	if !PackageJSONVersionUsableForGhost("1.2.3") {
		t.Fatal("exact semver should be usable")
	}
	if PackageJSONVersionUsableForGhost("^1.2.3") {
		t.Fatal("range should not be usable")
	}
	if PackageJSONVersionUsableForGhost("workspace:*") {
		t.Fatal("workspace protocol should not be usable")
	}
}
