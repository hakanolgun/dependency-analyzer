package npm

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/hakanolgun/dependency-analyzer/cli-go/internal/engine"
)

type npmVersionDoc struct {
	Dist *struct {
		Tarball string `json:"tarball"`
	} `json:"dist"`
}

// FetchTarballURL returns the registry tarball URL for an exact version.
func (c *Client) FetchTarballURL(ctx context.Context, name, version string) (string, error) {
	regRoot := registryBaseURL
	if c.RegistryBaseURL != "" {
		regRoot = strings.TrimSuffix(c.RegistryBaseURL, "/")
	}
	// Scoped: @scope/pkg -> @scope%2Fpkg in path segments
	escaped := url.PathEscape(name) + "/" + url.PathEscape(version)
	regURL := fmt.Sprintf("%s/%s", regRoot, escaped)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, regURL, nil)
	if err != nil {
		return "", err
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return "", fmt.Errorf("registry %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}
	var doc npmVersionDoc
	if err := json.NewDecoder(resp.Body).Decode(&doc); err != nil {
		return "", err
	}
	if doc.Dist == nil || doc.Dist.Tarball == "" {
		return "", errors.New("registry response missing dist.tarball")
	}
	return doc.Dist.Tarball, nil
}

// DownloadExtractPackage fetches a tarball and extracts the npm "package" folder to destDir.
func (c *Client) DownloadExtractPackage(ctx context.Context, tarballURL, destDir string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, tarballURL, nil)
	if err != nil {
		return err
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return fmt.Errorf("tarball %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}
	gz, err := gzip.NewReader(resp.Body)
	if err != nil {
		return err
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		if hdr.Name == "" || hdr.Typeflag == tar.TypeXGlobalHeader || hdr.Typeflag == tar.TypeXHeader {
			continue
		}
		rel := strings.TrimPrefix(hdr.Name, "./")
		if rel == "" {
			continue
		}
		parts := strings.SplitN(rel, "/", 2)
		if len(parts) < 2 {
			continue
		}
		rest := parts[1]
		target := filepath.Join(destDir, rest)
		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			f, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(hdr.Mode)&0o777)
			if err != nil {
				return err
			}
			if _, err := io.Copy(f, tr); err != nil {
				f.Close()
				return err
			}
			if err := f.Close(); err != nil {
				return err
			}
		case tar.TypeSymlink:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			if err := os.Symlink(hdr.Linkname, target); err != nil {
				return err
			}
		}
	}
	return nil
}

func analyzeDependencyGhost(ctx context.Context, c *Client, projectPath, depName, packageJSONSpec string) (engine.DependencyResult, string) {
	var res engine.DependencyResult
	res.Name = depName
	res.Version = packageJSONSpec

	ver, err := resolveVersionForGhost(projectPath, depName, packageJSONSpec)
	if err != nil {
		res.Error = "cannot resolve exact version (add package-lock.json, pnpm-lock.yaml, or yarn.lock, or pin an exact version in package.json)"
		return res, ""
	}
	res.Version = ver

	tarball, err := c.FetchTarballURL(ctx, depName, ver)
	if err != nil {
		res.Error = err.Error()
		return res, ""
	}

	tmpRoot, err := os.MkdirTemp("", "dependency-analyzer-npm-*")
	if err != nil {
		res.Error = err.Error()
		return res, ""
	}
	extractDir := filepath.Join(tmpRoot, "extracted")
	if err := os.MkdirAll(extractDir, 0o755); err != nil {
		_ = os.RemoveAll(tmpRoot)
		res.Error = err.Error()
		return res, ""
	}

	if err := c.DownloadExtractPackage(ctx, tarball, extractDir); err != nil {
		_ = os.RemoveAll(tmpRoot)
		res.Error = err.Error()
		return res, ""
	}

	metrics, err := collectMetrics(extractDir)
	if err != nil {
		_ = os.RemoveAll(tmpRoot)
		res.Error = err.Error()
		return res, ""
	}

	norm := engine.ComputeNormalized(metrics)
	res.Normalized = norm
	res.Score = engine.ToPercentageScore(norm)
	res.Label = engine.ScoreLabel(norm)
	res.Metrics = metrics
	return res, tmpRoot
}
