package engine

import (
	"regexp"
	"strings"
)

var (
	sshRegex = regexp.MustCompile(`^git@([^:]+):(.+)$`)
)

// NormalizeRepoURL converts SSH and other git protocols to HTTPS.
func NormalizeRepoURL(url string) string {
	if url == "" {
		return ""
	}

	res := url

	// Remove git+ prefix
	res = strings.TrimPrefix(res, "git+")

	// Remove .git suffix
	res = strings.TrimSuffix(res, ".git")

	// Handle ssh://
	res = strings.TrimPrefix(res, "ssh://")

	// Handle git@github.com:owner/repo
	if matches := sshRegex.FindStringSubmatch(res); len(matches) == 3 {
		host := matches[1]
		path := matches[2]
		res = "https://" + host + "/" + path
	} else if after, ok :=strings.CutPrefix(res, "git@"); ok  {
		// Handle git@host/path
		res = "https://" + strings.Replace(after, ":", "/", 1)
	}

	// Double check git://
	res = strings.Replace(res, "git://", "https://", 1)

	return res
}
