package main

import (
	"errors"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
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

// ReadLocalVersion runs '<app> --version' and parses the output as the version string.
func ReadLocalVersion() (int, int, int, error) {
	appName := filepath.Base(os.Args[0])
	cmd := exec.Command(appName, "--version")
	out, err := cmd.Output()
	if err != nil {
		return 0, 0, 0, err
	}
	version := strings.TrimSpace(string(out))
	return parseVersion(version)
}

// ReadRemoteVersion fetches the version number from the Version variable in main.go in the remote GitHub repository.
func ReadRemoteVersion() (int, int, int, error) {
	url := "https://raw.githubusercontent.com/fabriziotappero/timo/main/main.go"
	resp, err := http.Get(url)
	if err != nil {
		return 0, 0, 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 0, 0, 0, errors.New("failed to fetch remote main.go file")
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, 0, 0, err
	}

	// Use regex to find the Version variable assignment inside main.go
	re := regexp.MustCompile(`var Version\s*=\s*"(v[0-9]+\.[0-9]+\.[0-9]+)"`)
	matches := re.FindStringSubmatch(string(body))

	if len(matches) < 2 {
		return 0, 0, 0, errors.New("Version variable not found in remote main.go")
	}
	return parseVersion(matches[1])
}

// NewVersionAvailable returns true if the remote version is newer than the local version.
func NewVersionAvailable() (bool, error) {
	localMajor, localMinor, localPatch, err := ReadLocalVersion()
	if err != nil {
		return false, err
	}
	remoteMajor, remoteMinor, remotePatch, err := ReadRemoteVersion()
	if err != nil {
		return false, err
	}
	if remoteMajor > localMajor {
		return true, nil
	}
	if remoteMajor == localMajor && remoteMinor > localMinor {
		return true, nil
	}
	if remoteMajor == localMajor && remoteMinor == localMinor && remotePatch > localPatch {
		return true, nil
	}
	return false, nil
}
