package gomod

import (
	"strings"
	"testing"
)

func TestParseGoMod_Basic(t *testing.T) {
	content := `module github.com/user/project

go 1.21

require (
	github.com/gin-gonic/gin v1.9.1
	github.com/stretchr/testify v1.8.4
	golang.org/x/sync v0.3.0 // indirect
)
`
	result, err := ParseGoMod(content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ModuleName != "github.com/user/project" {
		t.Fatalf("got module name %q", result.ModuleName)
	}
	if result.GoVersion != "1.21" {
		t.Fatalf("got go version %q", result.GoVersion)
	}
	// Should have 2 deps (indirect is skipped)
	if len(result.Dependencies) != 2 {
		t.Fatalf("expected 2 deps, got %d", len(result.Dependencies))
	}
	if result.Dependencies[0].Path != "github.com/gin-gonic/gin" {
		t.Fatalf("got dep[0] path = %q", result.Dependencies[0].Path)
	}
	if result.Dependencies[0].Version != "v1.9.1" {
		t.Fatalf("got dep[0] version = %q", result.Dependencies[0].Version)
	}
}

func TestParseGoMod_SingleLineRequire(t *testing.T) {
	content := `module example.com/app

go 1.22

require github.com/pkg/errors v0.9.1
`
	result, err := ParseGoMod(content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Dependencies) != 1 {
		t.Fatalf("expected 1 dep, got %d", len(result.Dependencies))
	}
	if result.Dependencies[0].Path != "github.com/pkg/errors" {
		t.Fatalf("got path = %q", result.Dependencies[0].Path)
	}
}

func TestParseGoMod_NoModule(t *testing.T) {
	content := `go 1.21

require github.com/foo/bar v1.0.0
`
	_, err := ParseGoMod(content)
	if err == nil {
		t.Fatal("expected error for missing module declaration")
	}
}

func TestParseGoMod_NoDeps(t *testing.T) {
	content := `module example.com/empty

go 1.22
`
	result, err := ParseGoMod(content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Dependencies) != 0 {
		t.Fatalf("expected 0 deps, got %d", len(result.Dependencies))
	}
}

func TestEncodeModulePath(t *testing.T) {
	cases := []struct {
		input, want string
	}{
		{"github.com/Azure/go-sdk", "github.com/!azure/go-sdk"},
		{"github.com/user/repo", "github.com/user/repo"},
		{"github.com/BurntSushi/toml", "github.com/!burnt!sushi/toml"},
	}
	for _, tc := range cases {
		got := encodeModulePath(tc.input)
		if got != tc.want {
			t.Errorf("encodeModulePath(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestIsExternalImport(t *testing.T) {
	cases := []struct {
		path string
		want bool
	}{
		{"github.com/gin-gonic/gin", true},
		{"golang.org/x/sync", true},
		{"fmt", false},
		{"net/http", false},
		{"crypto/tls", false},
	}
	for _, tc := range cases {
		got := isExternalImport(tc.path)
		if got != tc.want {
			t.Errorf("isExternalImport(%q) = %v, want %v", tc.path, got, tc.want)
		}
	}
}

func TestComputeGoMetrics_NoCgo(t *testing.T) {
	stats := &sourceStats{
		sloc:            500,
		exportedFuncs:   5,
		exportedStructs: 2,
		exportedMethods: 4,
		goFileCount:     3,
	}
	m, conf := computeGoMetrics(stats)
	if m.Native != 0 {
		t.Errorf("expected native=0, got %v", m.Native)
	}
	if m.Volume != 0 {
		t.Errorf("expected volume=0 for <1000 SLOC, got %v", m.Volume)
	}
	if conf != "high" {
		t.Errorf("expected confidence=high, got %q", conf)
	}
}

func TestComputeGoMetrics_WithCgo(t *testing.T) {
	stats := &sourceStats{
		sloc:        5000,
		hasCgo:      true,
		goFileCount: 10,
	}
	m, _ := computeGoMetrics(stats)
	if m.Native != 1.0 {
		t.Errorf("expected native=1.0 with cgo, got %v", m.Native)
	}
	if m.Volume != 0.7 {
		t.Errorf("expected volume=0.7 for 5000 SLOC, got %v", m.Volume)
	}
}

func TestComputeGoMetrics_OsExecAddsEntanglement(t *testing.T) {
	stats := &sourceStats{
		sloc:        100,
		hasOsExec:   true,
		goFileCount: 1,
	}
	m, _ := computeGoMetrics(stats)
	if m.Entanglement < 0.1 {
		t.Errorf("expected entanglement >= 0.1 with os/exec, got %v", m.Entanglement)
	}
}

func TestComputeGoMetrics_ReflectAddsComplexity(t *testing.T) {
	stats := &sourceStats{
		sloc:        100,
		hasReflect:  true,
		goFileCount: 1,
	}
	m, _ := computeGoMetrics(stats)
	if m.LogicComplexity < 0.1 {
		t.Errorf("expected logicComplexity >= 0.1 with reflect, got %v", m.LogicComplexity)
	}
}

func TestComputeGoMetrics_LowConfidenceNoGoFiles(t *testing.T) {
	stats := &sourceStats{
		goFileCount: 0,
	}
	_, conf := computeGoMetrics(stats)
	if conf != "low" {
		t.Errorf("expected confidence=low with 0 go files, got %q", conf)
	}
}

func TestStripGoComments(t *testing.T) {
	src := `package foo
// single line comment
func Bar() { /* multi
line */ }
`
	cleaned := stripGoComments(src)
	if strings.Contains(cleaned, "single line comment") {
		t.Error("single-line comment not stripped")
	}
	if strings.Contains(cleaned, "multi\nline") {
		t.Error("multi-line comment not stripped")
	}
}

func TestComputeMaxBraceDepth(t *testing.T) {
	src := `func Foo() { if true { for { } } }`
	depth := computeMaxBraceDepth(src)
	if depth != 3 {
		t.Errorf("expected depth=3, got %d", depth)
	}
}


