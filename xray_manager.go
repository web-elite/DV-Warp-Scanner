package main

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"time"
)

const xrayRepository = "https://api.github.com/repos/XTLS/Xray-core"

var (
	xrayVersion = "unknown"
	bundledXrayVersion = "bundled"
	versionPattern = regexp.MustCompile("^v[0-9A-Za-z][0-9A-Za-z._-]*$")
)

type githubRelease struct {
	TagName string `json:"tag_name"`
	Assets []struct {
		Name string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
	} `json:"assets"`
}

func activeXrayVersion() string {
	if strings.TrimSpace(xrayVersion) == "" || xrayVersion == "unknown" {
		if data, err := os.ReadFile(filepath.Join(CORE_DIR, "version.txt")); err == nil && strings.TrimSpace(string(data)) != "" {
			return strings.TrimSpace(string(data))
		}
		if data, err := os.ReadFile(filepath.Join(CORE_DIR, "bundled-version.txt")); err == nil && strings.TrimSpace(string(data)) != "" {
			return strings.TrimSpace(string(data))
		}
		return "unknown"
	}
	return xrayVersion
}

func loadActiveXrayVersion(binary string) {
	bundledXrayVersion = "bundled"
	if data, err := os.ReadFile(filepath.Join(CORE_DIR, "bundled-version.txt")); err == nil && strings.TrimSpace(string(data)) != "" {
		bundledXrayVersion = strings.TrimSpace(string(data))
	}
	xrayVersion = bundledXrayVersion
	if data, err := os.ReadFile(filepath.Join(CORE_DIR, "version.txt")); err == nil && strings.TrimSpace(string(data)) != "" {
		xrayVersion = strings.TrimSpace(string(data))
	}
	candidate := filepath.Join(CORE_DIR, "versions", xrayVersion, binary)
	if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
		xrayPath = candidate
		return
	}
	xrayPath = filepath.Join(CORE_DIR, binary)
	if xrayVersion != bundledXrayVersion {
		xrayVersion = bundledXrayVersion
	}
}

func binaryNameForRuntime() string {
	if runtime.GOOS == "windows" {
		return "xray.exe"
	}
	return "xray"
}

func xrayAssetName() (string, error) {
	platform := runtime.GOOS
	if platform == "android" {
		platform = "linux"
	}
	if platform == "darwin" {
		platform = "macos"
	}
	arch := ""
	switch runtime.GOARCH {
	case "amd64":
		arch = "64"
	case "386":
		arch = "32"
	case "arm64":
		arch = "arm64-v8a"
	case "arm":
		if platform == "linux" {
			arch = "arm32-v5"
		}
	}
	if arch == "" || (platform != "linux" && platform != "windows" && platform != "macos") {
		return "", fmt.Errorf("Xray-core does not provide a supported build for %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	return fmt.Sprintf("Xray-%s-%s.zip", platform, arch), nil
}

func xrayManagerMenu() {
	for {
		fmt.Println()
		fmt.Println(fmtStr("Xray-core Manager", BLUE, true))
		fmt.Printf("Active version: %s\n", fmtStr(activeXrayVersion(), GREEN, true))
		fmt.Println("  0  Back")
		fmt.Println("  1  Check and install latest stable version")
		fmt.Println("  2  Install or activate a specific version")
		fmt.Println("  3  Show installed versions / switch version")
		fmt.Println("  4  Show active version")
		switch readMenuChoice("\nSelect an option: ") {
		case "0":
			return
		case "1":
			release, err := fetchLatestXrayRelease()
			if err != nil {
				failMessage("Could not check the latest Xray release: " + err.Error())
				continue
			}
			fmt.Printf("Latest stable version: %s\n", release.TagName)
			if release.TagName == activeXrayVersion() {
				successMessage("You are already using the latest stable version.")
				continue
			}
			if isXrayVersionInstalled(release.TagName) {
				if confirmAction("This version is already installed. Activate it? [y/N]: ") {
					if err := activateXrayVersion(release.TagName, binaryNameForRuntime()); err != nil {
						failMessage(err.Error())
					} else {
						successMessage("Activated Xray-core " + release.TagName + ".")
					}
				}
				continue
			}
			if confirmAction("Install and activate " + release.TagName + "? [y/N]: ") {
				if err := installAndActivateXray(release); err != nil {
					failMessage("Xray installation failed: " + err.Error())
				} else {
					successMessage("Xray-core " + release.TagName + " installed and activated.")
				}
			}
		case "2":
			tag := readMenuChoice("Enter exact Xray release tag (example: v26.11.9): ")
			if !versionPattern.MatchString(tag) {
				failMessage("Invalid release tag.")
				continue
			}
			if tag == activeXrayVersion() {
				successMessage("This version is already active.")
				continue
			}
			if isXrayVersionInstalled(tag) {
				if confirmAction("This version is already installed. Activate it? [y/N]: ") {
					if err := activateXrayVersion(tag, binaryNameForRuntime()); err != nil {
						failMessage(err.Error())
					} else {
						successMessage("Activated Xray-core " + tag + ".")
					}
				}
				continue
			}
			if confirmAction("Download, verify and activate " + tag + "? [y/N]: ") {
				release, err := fetchXrayRelease(tag)
				if err == nil {
					err = installAndActivateXray(release)
				}
				if err != nil {
					failMessage("Xray installation failed: " + err.Error())
				} else {
					successMessage("Xray-core " + tag + " installed and activated.")
				}
			}
		case "3":
			switchInstalledXrayVersion()
		case "4":
			fmt.Printf("\nActive Xray-core version: %s\n", activeXrayVersion())
		default:
			failMessage("Invalid option. Choose 0 to 4.")
		}
	}
}

func confirmAction(promptText string) bool {
	return strings.EqualFold(readMenuChoice(promptText), "y")
}

func isXrayVersionInstalled(tag string) bool {
	if tag == bundledXrayVersion {
		_, err := os.Stat(filepath.Join(CORE_DIR, binaryNameForRuntime()))
		return err == nil
	}
	info, err := os.Stat(filepath.Join(CORE_DIR, "versions", tag, binaryNameForRuntime()))
	return err == nil && !info.IsDir()
}

func fetchLatestXrayRelease() (githubRelease, error) {
	return requestXrayRelease(xrayRepository + "/releases/latest")
}

func fetchXrayRelease(tag string) (githubRelease, error) {
	return requestXrayRelease(xrayRepository + "/releases/tags/" + tag)
}

func requestXrayRelease(url string) (githubRelease, error) {
	client := &http.Client{Timeout: 25 * time.Second}
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return githubRelease{}, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "DV-Warp-Scanner")
	resp, err := client.Do(req)
	if err != nil {
		return githubRelease{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return githubRelease{}, fmt.Errorf("GitHub returned HTTP %d", resp.StatusCode)
	}
	var release githubRelease
	if err := json.NewDecoder(io.LimitReader(resp.Body, 4<<20)).Decode(&release); err != nil {
		return githubRelease{}, err
	}
	if !versionPattern.MatchString(release.TagName) {
		return githubRelease{}, fmt.Errorf("GitHub returned an invalid release tag")
	}
	return release, nil
}

func installAndActivateXray(release githubRelease) error {
	if !versionPattern.MatchString(release.TagName) {
		return fmt.Errorf("invalid Xray release tag")
	}
	assetName, err := xrayAssetName()
	if err != nil {
		return err
	}
	var archiveURL, digestURL string
	for _, asset := range release.Assets {
		switch asset.Name {
		case assetName:
			archiveURL = asset.BrowserDownloadURL
		case assetName + ".dgst":
			digestURL = asset.BrowserDownloadURL
		}
	}
	if archiveURL == "" || digestURL == "" {
		return fmt.Errorf("release %s does not contain %s and its checksum", release.TagName, assetName)
	}

	tempDir, err := os.MkdirTemp(CORE_DIR, "xray-download-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tempDir)

	archivePath := filepath.Join(tempDir, assetName)
	if err := downloadXrayFile(archiveURL, archivePath); err != nil {
		return err
	}
	digestData, err := downloadXrayBytes(digestURL, 1<<20)
	if err != nil {
		return err
	}
	match := regexp.MustCompile("(?i)[0-9a-f]{64}").Find(digestData)
	if len(match) != 64 {
		return fmt.Errorf("release checksum file does not contain a SHA-256 digest")
	}
	archiveData, err := os.ReadFile(archivePath)
	if err != nil {
		return err
	}
	sum := sha256.Sum256(archiveData)
	expected, err := hex.DecodeString(string(match))
	if err != nil || !equalBytes(sum[:], expected) {
		return fmt.Errorf("SHA-256 verification failed; archive was not installed")
	}

	versionDir := filepath.Join(CORE_DIR, "versions", release.TagName)
	stagingDir := versionDir + ".staging"
	_ = os.RemoveAll(stagingDir)
	if err := os.MkdirAll(stagingDir, 0755); err != nil {
		return err
	}
	if err := extractXrayArchive(archivePath, stagingDir); err != nil {
		_ = os.RemoveAll(stagingDir)
		return err
	}
	binaryPath := filepath.Join(stagingDir, binaryNameForRuntime())
	if info, err := os.Stat(binaryPath); err != nil || info.IsDir() {
		_ = os.RemoveAll(stagingDir)
		return fmt.Errorf("archive does not contain %s", binaryNameForRuntime())
	}
	if runtime.GOOS != "windows" {
		if err := os.Chmod(binaryPath, 0755); err != nil {
			_ = os.RemoveAll(stagingDir)
			return err
		}
	}
	if err := os.MkdirAll(filepath.Dir(versionDir), 0755); err != nil {
		_ = os.RemoveAll(stagingDir)
		return err
	}
	_ = os.RemoveAll(versionDir)
	if err := os.Rename(stagingDir, versionDir); err != nil {
		_ = os.RemoveAll(stagingDir)
		return err
	}
	return activateXrayVersion(release.TagName, binaryNameForRuntime())
}

func downloadXrayFile(url, destination string) error {
	data, err := downloadXrayBytes(url, 256<<20)
	if err != nil {
		return err
	}
	return os.WriteFile(destination, data, 0600)
}

func downloadXrayBytes(url string, maxBytes int64) ([]byte, error) {
	client := &http.Client{Timeout: 3 * time.Minute}
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "DV-Warp-Scanner")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download failed with HTTP %d", resp.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxBytes+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > maxBytes {
		return nil, fmt.Errorf("download exceeds the allowed size")
	}
	return data, nil
}

func extractXrayArchive(archivePath, destination string) error {
	reader, err := zip.OpenReader(archivePath)
	if err != nil {
		return err
	}
	defer reader.Close()
	root, err := filepath.Abs(destination)
	if err != nil {
		return err
	}
	for _, file := range reader.File {
		target := filepath.Join(destination, filepath.FromSlash(file.Name))
		absTarget, err := filepath.Abs(target)
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, absTarget)
		if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
			return fmt.Errorf("unsafe path in Xray archive")
		}
		if file.FileInfo().IsDir() {
			if err := os.MkdirAll(target, 0755); err != nil {
				return err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return err
		}
		src, err := file.Open()
		if err != nil {
			return err
		}
		mode := file.Mode().Perm()
		if mode == 0 {
			mode = 0644
		}
		dst, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, mode)
		if err != nil {
			_ = src.Close()
			return err
		}
		_, copyErr := io.Copy(dst, io.LimitReader(src, 128<<20))
		closeSrcErr := src.Close()
		closeDstErr := dst.Close()
		if copyErr != nil {
			return copyErr
		}
		if closeSrcErr != nil {
			return closeSrcErr
		}
		if closeDstErr != nil {
			return closeDstErr
		}
	}
	return nil
}

func activateXrayVersion(tag, binaryName string) error {
	path := filepath.Join(CORE_DIR, "versions", tag, binaryName)
	if tag == bundledXrayVersion {
		path = filepath.Join(CORE_DIR, binaryName)
	}
	if info, err := os.Stat(path); err != nil || info.IsDir() {
		return fmt.Errorf("version %s is not installed", tag)
	}
	tempPath := filepath.Join(CORE_DIR, "version.txt.tmp")
	versionPath := filepath.Join(CORE_DIR, "version.txt")
	if err := os.WriteFile(tempPath, []byte(tag+"\n"), 0644); err != nil {
		return err
	}
	// Windows does not replace an existing destination with os.Rename.
	_ = os.Remove(versionPath)
	if err := os.Rename(tempPath, versionPath); err != nil {
		_ = os.Remove(tempPath)
		return err
	}
	xrayVersion = tag
	xrayPath = path
	return nil
}

func switchInstalledXrayVersion() {
	dir := filepath.Join(CORE_DIR, "versions")
	entries, err := os.ReadDir(dir)
	if err != nil && !os.IsNotExist(err) {
		failMessage("Could not read installed Xray versions: " + err.Error())
		return
	}
	versions := make([]string, 0, len(entries)+1)
	for _, entry := range entries {
		if entry.IsDir() && versionPattern.MatchString(entry.Name()) {
			versions = append(versions, entry.Name())
		}
	}
	if isXrayVersionInstalled(bundledXrayVersion) {
		found := false
		for _, version := range versions {
			if version == bundledXrayVersion {
				found = true
				break
			}
		}
		if !found {
			versions = append(versions, bundledXrayVersion)
		}
	}
	sort.Strings(versions)
	if len(versions) == 0 {
		failMessage("No installed Xray versions were found.")
		return
	}
	fmt.Println("\nInstalled versions (* = active):")
	for i, version := range versions {
		marker := " "
		if version == activeXrayVersion() {
			marker = "*"
		}
		fmt.Printf("  %d  %s %s\n", i+1, marker, version)
	}
	choice := readMenuChoice("Enter version number (or 0 to cancel): ")
	if choice == "0" {
		return
	}
	var index int
	if _, err := fmt.Sscanf(choice, "%d", &index); err != nil || index < 1 || index > len(versions) {
		failMessage("Invalid selection.")
		return
	}
	if err := activateXrayVersion(versions[index-1], binaryNameForRuntime()); err != nil {
		failMessage("Could not activate version: " + err.Error())
		return
	}
	successMessage("Activated Xray-core " + versions[index-1] + ".")
}

func equalBytes(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	var diff byte
	for i := range a {
		diff |= a[i] ^ b[i]
	}
	return diff == 0
}
