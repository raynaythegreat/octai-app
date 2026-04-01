package utils

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/raynaythegreat/octai-app/pkg/config"
)

// GetPicoclawHome returns the octai home directory.
// Priority: $OCTAI_HOME > ~/.octai
func GetPicoclawHome() string {
	if home := os.Getenv(config.EnvHome); home != "" {
		return home
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".octai")
}

// GetAIBHQHome returns the octai home directory (alias for GetPicoclawHome).
func GetAIBHQHome() string {
	return GetPicoclawHome()
}

// GetDefaultConfigPath returns the default path to the octai config file.
func GetDefaultConfigPath() string {
	if configPath := os.Getenv(config.EnvConfig); configPath != "" {
		return configPath
	}
	return filepath.Join(GetPicoclawHome(), "config.json")
}

// FindPicoclawBinary locates the octai executable.
// Search order:
//  1. OCTAI_BINARY environment variable (explicit override)
//  2. Same directory as the current executable (tries multiple binary names)
//  3. Falls back to "octai" and relies on $PATH
func FindPicoclawBinary() string {
	candidates := []string{"octai", "octai-backend", "octai-app"}
	if runtime.GOOS == "windows" {
		for i := range candidates {
			candidates[i] += ".exe"
		}
	}

	if p := os.Getenv(config.EnvBinary); p != "" {
		if info, _ := os.Stat(p); info != nil && !info.IsDir() {
			return p
		}
	}

	if exe, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exe)
		for _, name := range candidates {
			candidate := filepath.Join(exeDir, name)
			if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
				return candidate
			}
		}
	}

	return candidates[0]
}

// GetLocalIP returns the local IP address of the machine.
func GetLocalIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ""
	}
	for _, a := range addrs {
		if ipnet, ok := a.(*net.IPNet); ok && !ipnet.IP.IsLoopback() && ipnet.IP.To4() != nil {
			return ipnet.IP.String()
		}
	}
	return ""
}

// OpenBrowser automatically opens the given URL in the default browser.
func OpenBrowser(url string) error {
	switch runtime.GOOS {
	case "linux":
		return exec.Command("xdg-open", url).Start()
	case "windows":
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		return exec.Command("open", url).Start()
	default:
		return fmt.Errorf("unsupported platform")
	}
}
