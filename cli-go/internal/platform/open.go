package platform

import (
	"os/exec"
	"runtime"
)

func OpenBrowser(path string) error {
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("open", path).Run()
	case "windows":
		return exec.Command("cmd", "/c", "start", path).Run()
	default:
		return exec.Command("xdg-open", path).Run()
	}
}
