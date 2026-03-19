package update

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	runtimepkg "runtime"
	"strconv"
	"strings"
	"sync"
	"time"
	"watools/config"
)

const (
	repositorySlug     = "a31521424/watools"
	latestManifestURL  = "https://github.com/a31521424/watools/releases/latest/download/watools_latest.json"
	defaultReleasePage = "https://github.com/a31521424/watools/releases"
	httpTimeout        = 45 * time.Second
)

type ReleaseManifest struct {
	Version     string                   `json:"version"`
	ReleaseTag  string                   `json:"releaseTag"`
	ReleaseName string                   `json:"releaseName"`
	ReleaseURL  string                   `json:"releaseUrl"`
	PublishedAt string                   `json:"publishedAt"`
	Notes       string                   `json:"notes"`
	GeneratedAt string                   `json:"generatedAt"`
	Platforms   map[string]ManifestAsset `json:"platforms"`
}

type ManifestAsset struct {
	OS            string `json:"os"`
	Arch          string `json:"arch"`
	AssetName     string `json:"assetName"`
	URL           string `json:"url"`
	SHA256        string `json:"sha256"`
	Size          int64  `json:"size"`
	InstallerType string `json:"installerType"`
}

type UpdateInfo struct {
	CurrentVersion        string `json:"currentVersion"`
	LatestVersion         string `json:"latestVersion"`
	ReleaseTag            string `json:"releaseTag"`
	ReleaseName           string `json:"releaseName"`
	ReleaseURL            string `json:"releaseUrl"`
	PublishedAt           string `json:"publishedAt"`
	Notes                 string `json:"notes"`
	PlatformKey           string `json:"platformKey"`
	PlatformLabel         string `json:"platformLabel"`
	AssetName             string `json:"assetName"`
	AssetURL              string `json:"assetUrl"`
	AssetSHA256           string `json:"assetSha256"`
	AssetSize             int64  `json:"assetSize"`
	InstallerType         string `json:"installerType"`
	HasUpdate             bool   `json:"hasUpdate"`
	DownloadedPath        string `json:"downloadedPath"`
	RequiresManualInstall bool   `json:"requiresManualInstall"`
	Message               string `json:"message"`
}

type InstallResult struct {
	Status                string `json:"status"`
	Message               string `json:"message"`
	OpenPath              string `json:"openPath"`
	ShouldQuit            bool   `json:"shouldQuit"`
	RequiresManualInstall bool   `json:"requiresManualInstall"`
}

type Service struct {
	client *http.Client
}

var (
	serviceOnce sync.Once
	serviceInst *Service
)

func GetService() *Service {
	serviceOnce.Do(func() {
		serviceInst = &Service{
			client: &http.Client{Timeout: httpTimeout},
		}
	})
	return serviceInst
}

func (s *Service) Check(ctx context.Context) (UpdateInfo, error) {
	info, _, err := s.resolveUpdate(ctx)
	if err != nil {
		return UpdateInfo{}, err
	}

	if !info.HasUpdate {
		info.Message = "当前已是最新版本"
		return info, nil
	}

	if info.DownloadedPath != "" {
		info.Message = "检测到新版本，安装包已下载"
		return info, nil
	}

	info.Message = "检测到可用更新"
	return info, nil
}

func (s *Service) Download(ctx context.Context) (UpdateInfo, error) {
	info, asset, err := s.resolveUpdate(ctx)
	if err != nil {
		return UpdateInfo{}, err
	}
	if !info.HasUpdate {
		info.Message = "当前已是最新版本"
		return info, nil
	}

	targetPath := updateAssetPath(info.LatestVersion, asset.AssetName)
	if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
		return UpdateInfo{}, fmt.Errorf("failed to create update directory: %w", err)
	}

	if existingPath, err := verifyExistingAsset(targetPath, asset.SHA256); err == nil && existingPath != "" {
		info.DownloadedPath = existingPath
		info.RequiresManualInstall = requiresManualInstall(asset.InstallerType)
		info.Message = "更新包已存在，跳过重复下载"
		return info, nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, asset.URL, nil)
	if err != nil {
		return UpdateInfo{}, fmt.Errorf("failed to create download request: %w", err)
	}
	req.Header.Set("User-Agent", updateUserAgent())
	req.Header.Set("Accept", "application/octet-stream")

	resp, err := s.client.Do(req)
	if err != nil {
		return UpdateInfo{}, fmt.Errorf("failed to download release asset: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return UpdateInfo{}, fmt.Errorf("download request failed with status %s", resp.Status)
	}

	tempPath := targetPath + ".tmp"
	file, err := os.Create(tempPath)
	if err != nil {
		return UpdateInfo{}, fmt.Errorf("failed to create temp asset file: %w", err)
	}

	hasher := sha256.New()
	writer := io.MultiWriter(file, hasher)
	if _, err := io.Copy(writer, resp.Body); err != nil {
		_ = file.Close()
		_ = os.Remove(tempPath)
		return UpdateInfo{}, fmt.Errorf("failed to write release asset: %w", err)
	}
	if err := file.Close(); err != nil {
		_ = os.Remove(tempPath)
		return UpdateInfo{}, fmt.Errorf("failed to close release asset: %w", err)
	}

	if asset.SHA256 != "" {
		sum := hex.EncodeToString(hasher.Sum(nil))
		if !strings.EqualFold(sum, asset.SHA256) {
			_ = os.Remove(tempPath)
			return UpdateInfo{}, fmt.Errorf("checksum mismatch: expected %s, got %s", asset.SHA256, sum)
		}
	}

	if err := os.Rename(tempPath, targetPath); err != nil {
		_ = os.Remove(tempPath)
		return UpdateInfo{}, fmt.Errorf("failed to finalize release asset: %w", err)
	}

	info.DownloadedPath = targetPath
	info.RequiresManualInstall = requiresManualInstall(asset.InstallerType)
	info.Message = "更新包下载完成"
	return info, nil
}

func (s *Service) Install(downloadedPath string) (InstallResult, error) {
	if strings.TrimSpace(downloadedPath) == "" {
		return InstallResult{}, fmt.Errorf("update asset path is empty")
	}
	if _, err := os.Stat(downloadedPath); err != nil {
		return InstallResult{}, fmt.Errorf("update asset is not available: %w", err)
	}

	result, err := installUpdateAsset(downloadedPath)
	if err != nil {
		return InstallResult{}, err
	}
	if result.OpenPath == "" {
		result.OpenPath = downloadedPath
	}
	return result, nil
}

func (s *Service) resolveUpdate(ctx context.Context) (UpdateInfo, ManifestAsset, error) {
	manifest, err := s.fetchManifest(ctx)
	if err != nil {
		return UpdateInfo{}, ManifestAsset{}, err
	}

	platformKey, platformLabel, err := currentPlatformDescriptor()
	if err != nil {
		return UpdateInfo{}, ManifestAsset{}, err
	}

	asset, ok := manifest.Platforms[platformKey]
	if !ok {
		return UpdateInfo{}, ManifestAsset{}, fmt.Errorf("release manifest does not contain an asset for %s", platformKey)
	}

	currentVersion := config.ProjectVersion()
	compareResult, err := compareVersions(manifest.Version, currentVersion)
	if err != nil {
		return UpdateInfo{}, ManifestAsset{}, err
	}

	info := UpdateInfo{
		CurrentVersion: currentVersion,
		LatestVersion:  manifest.Version,
		ReleaseTag:     manifest.ReleaseTag,
		ReleaseName:    manifest.ReleaseName,
		ReleaseURL:     manifest.ReleaseURL,
		PublishedAt:    manifest.PublishedAt,
		Notes:          manifest.Notes,
		PlatformKey:    platformKey,
		PlatformLabel:  platformLabel,
		AssetName:      asset.AssetName,
		AssetURL:       asset.URL,
		AssetSHA256:    asset.SHA256,
		AssetSize:      asset.Size,
		InstallerType:  asset.InstallerType,
		HasUpdate:      compareResult > 0,
	}
	if info.ReleaseURL == "" {
		info.ReleaseURL = defaultReleasePage
	}

	downloadedPath := updateAssetPath(info.LatestVersion, asset.AssetName)
	if existingPath, err := verifyExistingAsset(downloadedPath, asset.SHA256); err == nil && existingPath != "" {
		info.DownloadedPath = existingPath
		info.RequiresManualInstall = requiresManualInstall(asset.InstallerType)
	}

	return info, asset, nil
}

func (s *Service) fetchManifest(ctx context.Context) (ReleaseManifest, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, latestManifestURL, nil)
	if err != nil {
		return ReleaseManifest{}, fmt.Errorf("failed to create manifest request: %w", err)
	}
	req.Header.Set("User-Agent", updateUserAgent())
	req.Header.Set("Accept", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return ReleaseManifest{}, fmt.Errorf("failed to fetch update manifest: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return ReleaseManifest{}, fmt.Errorf("manifest request failed with status %s", resp.Status)
	}

	var manifest ReleaseManifest
	if err := json.NewDecoder(resp.Body).Decode(&manifest); err != nil {
		return ReleaseManifest{}, fmt.Errorf("failed to decode update manifest: %w", err)
	}
	if manifest.Version == "" {
		return ReleaseManifest{}, fmt.Errorf("update manifest is missing version")
	}
	if len(manifest.Platforms) == 0 {
		return ReleaseManifest{}, fmt.Errorf("update manifest is missing platform assets")
	}
	return manifest, nil
}

func currentPlatformDescriptor() (string, string, error) {
	switch runtimepkg.GOOS {
	case "windows":
		if runtimepkg.GOARCH != "amd64" {
			return "", "", fmt.Errorf("unsupported windows architecture: %s", runtimepkg.GOARCH)
		}
		return "windows-amd64", "Windows x64", nil
	case "darwin":
		return "darwin-universal", "macOS Universal", nil
	default:
		return "", "", fmt.Errorf("self-update is not supported on %s", runtimepkg.GOOS)
	}
}

func requiresManualInstall(installerType string) bool {
	switch installerType {
	case "app-zip":
		return runtimepkg.GOOS != "darwin"
	case "nsis":
		return false
	default:
		return false
	}
}

func updateAssetPath(version string, assetName string) string {
	return filepath.Join(config.ProjectCacheDir(), "updates", sanitizePathPart(version), assetName)
}

func sanitizePathPart(value string) string {
	replacer := strings.NewReplacer("/", "-", "\\", "-", ":", "-", " ", "-")
	sanitized := replacer.Replace(strings.TrimSpace(value))
	if sanitized == "" {
		return "unknown"
	}
	return sanitized
}

func verifyExistingAsset(path string, expectedSHA string) (string, error) {
	stat, err := os.Stat(path)
	if err != nil {
		return "", err
	}
	if stat.IsDir() {
		return "", fmt.Errorf("update asset path %s is a directory", path)
	}
	if expectedSHA == "" {
		return path, nil
	}
	sum, err := checksumFile(path)
	if err != nil {
		return "", err
	}
	if !strings.EqualFold(sum, expectedSHA) {
		return "", fmt.Errorf("checksum mismatch for cached asset")
	}
	return path, nil
}

func checksumFile(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hasher.Sum(nil)), nil
}

type semanticVersion struct {
	core       []int
	prerelease []string
}

func compareVersions(nextVersion string, currentVersion string) (int, error) {
	next, err := parseVersion(nextVersion)
	if err != nil {
		return 0, fmt.Errorf("invalid latest version %q: %w", nextVersion, err)
	}
	current, err := parseVersion(currentVersion)
	if err != nil {
		return 0, fmt.Errorf("invalid current version %q: %w", currentVersion, err)
	}

	for i := 0; i < len(next.core) || i < len(current.core); i++ {
		nextSegment := 0
		if i < len(next.core) {
			nextSegment = next.core[i]
		}
		currentSegment := 0
		if i < len(current.core) {
			currentSegment = current.core[i]
		}

		switch {
		case nextSegment > currentSegment:
			return 1, nil
		case nextSegment < currentSegment:
			return -1, nil
		}
	}

	return comparePrereleaseIdentifiers(next.prerelease, current.prerelease), nil
}

func parseVersion(raw string) (semanticVersion, error) {
	version := strings.TrimSpace(raw)
	version = strings.TrimPrefix(version, "v")
	version = strings.SplitN(version, "+", 2)[0]
	if version == "" {
		return semanticVersion{}, fmt.Errorf("empty version")
	}

	parts := strings.SplitN(version, "-", 2)
	coreRaw := parts[0]
	if coreRaw == "" {
		return semanticVersion{}, fmt.Errorf("missing version core")
	}

	coreParts := strings.Split(coreRaw, ".")
	core := make([]int, 0, len(coreParts))
	for _, segment := range coreParts {
		if segment == "" {
			return semanticVersion{}, fmt.Errorf("invalid core segment in %q", raw)
		}
		value, err := strconv.Atoi(segment)
		if err != nil {
			return semanticVersion{}, fmt.Errorf("invalid numeric segment %q", segment)
		}
		core = append(core, value)
	}

	parsed := semanticVersion{core: core}
	if len(parts) == 2 {
		for _, segment := range strings.Split(parts[1], ".") {
			if segment == "" {
				return semanticVersion{}, fmt.Errorf("invalid prerelease segment in %q", raw)
			}
			parsed.prerelease = append(parsed.prerelease, segment)
		}
	}

	return parsed, nil
}

func comparePrereleaseIdentifiers(next []string, current []string) int {
	switch {
	case len(next) == 0 && len(current) == 0:
		return 0
	case len(next) == 0:
		return 1
	case len(current) == 0:
		return -1
	}

	maxLength := len(next)
	if len(current) > maxLength {
		maxLength = len(current)
	}

	for i := 0; i < maxLength; i++ {
		if i >= len(next) {
			return -1
		}
		if i >= len(current) {
			return 1
		}

		nextValue, nextErr := strconv.Atoi(next[i])
		currentValue, currentErr := strconv.Atoi(current[i])
		switch {
		case nextErr == nil && currentErr == nil:
			switch {
			case nextValue > currentValue:
				return 1
			case nextValue < currentValue:
				return -1
			}
		case nextErr == nil:
			return -1
		case currentErr == nil:
			return 1
		default:
			switch {
			case next[i] > current[i]:
				return 1
			case next[i] < current[i]:
				return -1
			}
		}
	}

	return 0
}

func updateUserAgent() string {
	return fmt.Sprintf("%s/%s (%s)", config.ProjectName(), config.ProjectVersion(), repositorySlug)
}
