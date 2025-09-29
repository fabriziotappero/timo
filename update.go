package main

import (
	"archive/zip"
	"debug/buildinfo"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
)

// parses a version string like "v1.2.3" and returns major, minor, patch.
func parseVersion(ver string) (int, int, int, error) {
	ver = strings.TrimSpace(ver)
	ver = strings.TrimPrefix(ver, "v")
	parts := strings.Split(ver, ".")
	if len(parts) != 3 {
		return 0, 0, 0, errors.New("invalid version format")
	}
	major, err1 := strconv.Atoi(parts[0])
	minor, err2 := strconv.Atoi(parts[1])
	patch, err3 := strconv.Atoi(parts[2])
	if err1 != nil || err2 != nil || err3 != nil {
		return 0, 0, 0, errors.New("invalid version number")
	}
	return major, minor, patch, nil
}

// extract version from the binary file using buildinfo method
func ReadLocalVersion() (int, int, int, error) {
	exePath, err := os.Executable()
	if err != nil {
		slog.Error("Failed to get executable path", "error", err)
		return 0, 0, 0, err
	}
	info, err := buildinfo.ReadFile(exePath)
	if err != nil {
		slog.Error("Failed to read build info from executable", "error", err)
		return 0, 0, 0, err
	}

	// Look for version in the -ldflags setting value (e.g. '-X main.version=v0.2.0 ...')
	for _, setting := range info.Settings {
		// slog.Info(fmt.Sprintf("%s: %s", setting.Key, setting.Value))
		if setting.Key == "-ldflags" && strings.Contains(setting.Value, "main.version=") {
			re := regexp.MustCompile(`main\.version=(v?\d+\.\d+\.\d+)`)
			match := re.FindStringSubmatch(setting.Value)
			if len(match) > 1 {
				version := match[1]
				slog.Info("Extracted version from ldflags", "version", version)
				return parseVersion(version)
			}
		}
	}
	slog.Warn("Version not found in binary file")
	return 0, 0, 0, errors.New("version not found in binary file")
}

// fetches the software version from the VERSION variable in
// the remote build.sh file in the GitHub repository.
func ReadRemoteVersion() (int, int, int, error) {
	url := "https://raw.githubusercontent.com/fabriziotappero/timo/main/build.sh"
	resp, err := http.Get(url)
	if err != nil {
		return 0, 0, 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 0, 0, 0, errors.New("failed to fetch remote build.sh file")
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, 0, 0, err
	}
	// Extract VERSION=... from the build.sh file
	lines := strings.Split(string(body), "\n")
	var version string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "VERSION=") {
			version = strings.TrimPrefix(line, "VERSION=")
			version = strings.Trim(version, "\"'")
			break
		}
	}
	if version == "" {
		return 0, 0, 0, errors.New("VERSION variable not found in remote build.sh")
	}
	return parseVersion(version)
}

// returns true if the remote version in github is newer than the local version.
// local version is extracted from the binary file using buildinfo method
func NewVersionAvailable() (bool, error) {
	localMajor, localMinor, localPatch, err := ReadLocalVersion()
	if err != nil {
		slog.Warn("Failed to read local version", "error", err)
		return false, err
	}
	remoteMajor, remoteMinor, remotePatch, err := ReadRemoteVersion()
	if err != nil {
		slog.Warn("Failed to read remote version", "error", err)
		return false, err
	}
	slog.Info("Version check", "local", fmt.Sprintf("%d.%d.%d", localMajor, localMinor, localPatch), "remote", fmt.Sprintf("%d.%d.%d", remoteMajor, remoteMinor, remotePatch))
	if remoteMajor > localMajor {
		slog.Info("New version available from remote: remote major > local major")
		return true, nil
	}
	if remoteMajor == localMajor && remoteMinor > localMinor {
		slog.Info("New version available from remote: remote minor > local minor")
		return true, nil
	}
	if remoteMajor == localMajor && remoteMinor == localMinor && remotePatch > localPatch {
		slog.Info("New version available from remote: remote patch > local patch")
		return true, nil
	}
	slog.Info("No new version available from remote")
	return false, nil
}

// checks if 'chrome' or 'chromium' is available in PATH or in common installation directories
func IsChromiumAvailable() bool {
	execPath := FindChromiumExecutable()
	return execPath != ""
}

// finds and returns the full path to Chrome/Chromium executable, or empty string if not found
func FindChromiumExecutable() string {
	// First check PATH
	candidates := []string{"chrome", "chromium", "chrome.exe", "chromium.exe", "google-chrome", "google-chrome-stable"}
	for _, name := range candidates {
		if path, err := exec.LookPath(name); err == nil {
			slog.Info("Found Chrome/Chromium browser executable in PATH", "path", path)
			return path
		}
	}

	// Then check common installation locations on Windows
	if runtime.GOOS == "windows" {
		commonExecutables := []string{
			"C:\\Program Files\\Google\\Chrome\\Application\\chrome.exe",
			"C:\\Program Files (x86)\\Google\\Chrome\\Application\\chrome.exe",
			"C:\\Program Files\\Chromium\\Application\\chrome.exe",
			"C:\\Program Files (x86)\\Chromium\\Application\\chrome.exe",
		}

		for _, execPath := range commonExecutables {
			if _, err := os.Stat(execPath); err == nil {
				slog.Info("Found Chrome/Chromium browser executable in common location", "path", execPath)
				return execPath
			}
		}
	}

	slog.Warn("No Chrome/Chromium browser found in PATH or common locations")
	return ""
}

// downloads a portable Chromium ZIP to the OS temp folder (Windows/Linux)
// get version 787553 from 2020 (older but small)
func DownloadChromium() error {
	var url string
	if runtime.GOOS == "windows" {
		url = "https://www.googleapis.com/download/storage/v1/b/chromium-browser-snapshots/o/Win%2F787553%2Fchrome-win.zip?generation=1594518380823178&alt=media"
	} else if runtime.GOOS == "linux" {
		url = "https://www.googleapis.com/download/storage/v1/b/chromium-browser-snapshots/o/Linux_x64%2F785537%2Fchrome-linux.zip?generation=1594074384864843&alt=media"
	} else {
		slog.Error("Automatic Chromium download not supported on this OS")
		return fmt.Errorf("unsupported OS")
	}

	tempDir := os.TempDir()
	zipPath := filepath.Join(tempDir, "chromium.zip")

	out, err := os.Create(zipPath)
	if err != nil {
		slog.Error("Failed to create zip file", "error", err)
		return err
	}
	defer out.Close()

	slog.Info("Downloading Chromium...", "url", url)
	resp, err := http.Get(url)
	if err != nil {
		slog.Error("Failed to download Chromium", "error", err)
		return err
	}
	defer resp.Body.Close()
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		slog.Error("Failed to save Chromium zip", "error", err)
		return err
	}
	slog.Info("Chromium ZIP downloaded", "path", zipPath)
	return nil
}

// unzip extracts a zip archive to a destination directory
func unzip(src, dest string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()
	os.MkdirAll(dest, 0755)
	for _, f := range r.File {
		fpath := filepath.Join(dest, f.Name)
		if f.FileInfo().IsDir() {
			os.MkdirAll(fpath, f.Mode())
			continue
		}
		if err := os.MkdirAll(filepath.Dir(fpath), 0755); err != nil {
			return err
		}
		outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return err
		}
		rc, err := f.Open()
		if err != nil {
			outFile.Close()
			return err
		}
		_, err = io.Copy(outFile, rc)
		outFile.Close()
		rc.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

// extracts Chromium.zip from the OS temp folder to the user config directory
// which in Linux is  ~/.config and in Windows is C:\Users\yourname\AppData\Roaming
func InstallCustomChromium() error {

	configDir, err := os.UserConfigDir()
	if err != nil {
		slog.Error("No config dir", "err", err)
		return err
	}
	chromiumDir := filepath.Join(configDir, "timo", "chromium")
	os.MkdirAll(chromiumDir, 0755)

	zipPath := filepath.Join(os.TempDir(), "chromium.zip")
	if _, err := os.Stat(zipPath); err != nil {
		slog.Error("chromium.zip not found in OS temp folder", "err", err)
		return err
	}
	if err = unzip(zipPath, chromiumDir); err != nil {
		slog.Error("Unzip fail", "err", err)
		return err
	}
	slog.Info("Chromium extracted", "dir", chromiumDir)

	return nil
}

// finds a possible custom Chromium executable and returns its path for chromedp usage
// it searches in the user config directory ~/.config (Linux) or C:\Users\yourname\AppData\Roaming (Windows)
func GetCustomChromiumToPath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		slog.Error("No config dir", "err", err)
		return "", err
	}
	chromiumDir := filepath.Join(configDir, "timo", "chromium")
	var chromiumExe string
	switch runtime.GOOS {
	case "windows":
		chromiumExe = filepath.Join(chromiumDir, "chrome-win", "chrome.exe")
	case "linux":
		chromiumExe = filepath.Join(chromiumDir, "chrome-linux", "chrome")
	default:
		return "", fmt.Errorf("unsupported OS: %s", runtime.GOOS)
	}
	if _, err := os.Stat(chromiumExe); os.IsNotExist(err) {
		slog.Info("Chromium/Chrome executable not found at " + chromiumExe)
		return "", fmt.Errorf("chromium executable not found at %s", chromiumExe)
	}
	slog.Info("Chromium/Chrome is now available.", "path", chromiumExe)

	return chromiumExe, nil
}
