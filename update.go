package main

import (
	"debug/buildinfo"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
)

// parseVersion parses a version string like "v1.2.3" and returns major, minor, patch.
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

// ReadRemoteVersion fetches the version number from the VERSION variable in
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

// NewVersionAvailable returns true if the remote version is newer than the local version.
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
