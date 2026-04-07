package npm

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

// isPackageBinEntry is true for files directly under <package>/bin/ (npm bin layout).
func isPackageBinEntry(packageRoot, filePath string) bool {
	rel, err := filepath.Rel(packageRoot, filePath)
	if err != nil || rel == "." || strings.HasPrefix(rel, "..") {
		return false
	}
	parts := strings.Split(rel, string(filepath.Separator))
	return len(parts) == 2 && strings.EqualFold(parts[0], "bin")
}

// fileLooksLikeNativeExecutable detects real binaries under bin/ while avoiding
// false positives for Node/shell/Python CLIs (#!/usr/bin/env node, etc.).
func fileLooksLikeNativeExecutable(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()

	buf := make([]byte, 512)
	n, err := f.Read(buf)
	if err != nil && n == 0 {
		return false
	}
	buf = buf[:n]

	if isELFHeader(buf) || isMachOHeader(buf) || isPEHeader(buf) {
		return true
	}

	if bytes.IndexByte(buf, 0) >= 0 {
		return true
	}

	if bytes.HasPrefix(buf, []byte("#!")) {
		line := shebangFirstLine(buf)
		if interpreterShebangNegatesNative(line) {
			return false
		}
		// Unknown interpreter: treat as non-native (avoid marking odd scripts as native).
		return false
	}

	return !utf8.Valid(buf)
}

func shebangFirstLine(buf []byte) string {
	idx := bytes.IndexByte(buf, '\n')
	if idx < 0 {
		return string(buf)
	}
	line := buf[:idx]
	line = bytes.TrimSuffix(line, []byte{'\r'})
	return string(line)
}

func interpreterShebangNegatesNative(line string) bool {
	s := strings.TrimSpace(strings.ToLower(line))
	if !strings.HasPrefix(s, "#!") {
		return false
	}
	rest := strings.TrimSpace(s[2:])
	if rest == "" {
		return false
	}

	if i := strings.Index(rest, "env "); i >= 0 {
		tail := strings.TrimSpace(rest[i+4:])
		cmd := tail
		if sp := strings.IndexByte(tail, ' '); sp >= 0 {
			cmd = tail[:sp]
		}
		return isScriptInterpreterName(filepath.Base(cmd))
	}

	fields := strings.Fields(rest)
	if len(fields) == 0 {
		return false
	}
	return isScriptInterpreterName(filepath.Base(fields[0]))
}

func isScriptInterpreterName(base string) bool {
	base = strings.ToLower(strings.TrimSpace(base))
	switch base {
	case "node", "nodejs", "python", "python3", "python2", "ruby", "perl", "php",
		"bash", "sh", "zsh", "fish", "dash", "ksh", "csh", "tcsh",
		"bun", "deno", "npx", "yarn", "pnpm", "corepack":
		return true
	default:
		return false
	}
}

func isELFHeader(b []byte) bool {
	return len(b) >= 4 && b[0] == 0x7f && b[1] == 'E' && b[2] == 'L' && b[3] == 'F'
}

func isPEHeader(b []byte) bool {
	return len(b) >= 2 && b[0] == 'M' && b[1] == 'Z'
}

func isMachOHeader(b []byte) bool {
	if len(b) < 4 {
		return false
	}
	switch string(b[0:4]) {
	case "\xFE\xED\xFA\xCE", "\xFE\xED\xFA\xCF": // BE 32/64
		return true
	case "\xCE\xFA\xED\xFE", "\xCF\xFA\xED\xFE": // LE 32/64
		return true
	case "\xCA\xFE\xBA\xBE": // fat
		return true
	default:
		return false
	}
}
