package npm

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIsPackageBinEntry(t *testing.T) {
	root := t.TempDir()
	binFile := filepath.Join(root, "bin", "tool")
	if err := os.MkdirAll(filepath.Dir(binFile), 0o755); err != nil {
		t.Fatal(err)
	}
	if !isPackageBinEntry(root, binFile) {
		t.Fatal("expected bin/tool to match")
	}
	if isPackageBinEntry(root, filepath.Join(root, "bin", "sub", "x")) {
		t.Fatal("nested bin path should not match")
	}
	if isPackageBinEntry(root, filepath.Join(root, "lib", "x.js")) {
		t.Fatal("lib file should not match")
	}
}

func TestFileLooksLikeNativeExecutable_NodeShebang(t *testing.T) {
	root := t.TempDir()
	p := filepath.Join(root, "cli")
	content := "#!/usr/bin/env node\nconsole.log(1)\n"
	if err := os.WriteFile(p, []byte(content), 0o755); err != nil {
		t.Fatal(err)
	}
	if fileLooksLikeNativeExecutable(p) {
		t.Fatal("node shebang must not be native")
	}
}

func TestFileLooksLikeNativeExecutable_ELF(t *testing.T) {
	root := t.TempDir()
	p := filepath.Join(root, "app")
	elf := []byte{0x7f, 'E', 'L', 'F', 2, 1, 1, 0}
	if err := os.WriteFile(p, elf, 0o755); err != nil {
		t.Fatal(err)
	}
	if !fileLooksLikeNativeExecutable(p) {
		t.Fatal("ELF header should be native")
	}
}

func TestFileLooksLikeNativeExecutable_NullByte(t *testing.T) {
	root := t.TempDir()
	p := filepath.Join(root, "blob")
	if err := os.WriteFile(p, []byte("foo\x00bar"), 0o644); err != nil {
		t.Fatal(err)
	}
	if !fileLooksLikeNativeExecutable(p) {
		t.Fatal("null byte prefix should suggest binary")
	}
}

func TestFileLooksLikeNativeExecutable_PlainUTF8NoShebang(t *testing.T) {
	root := t.TempDir()
	p := filepath.Join(root, "txt")
	if err := os.WriteFile(p, []byte("hello world"), 0o644); err != nil {
		t.Fatal(err)
	}
	if fileLooksLikeNativeExecutable(p) {
		t.Fatal("plain text without shebang should not be native")
	}
}

func TestInterpreterShebangNegatesNative(t *testing.T) {
	if !interpreterShebangNegatesNative("#!/usr/bin/env bash") {
		t.Fatal("bash should negate")
	}
	if !interpreterShebangNegatesNative("#!/bin/sh") {
		t.Fatal("sh should negate")
	}
	if interpreterShebangNegatesNative("#!/opt/my-native-tool") {
		t.Fatal("unknown path should not negate via interpreter list")
	}
}
