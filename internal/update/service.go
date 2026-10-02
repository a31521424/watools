package update

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
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
	client     *http.Client
	mu         sync.Mutex
	downloadMu sync.Mutex
	verified   map[string]ManifestAsset
}

var (
	serviceOnce sync.Once
	serviceInst *Service
)

func GetService() *Service {
	serviceOnce.Do(func() {
		serviceInst = &Service{
			client:   &http.Client{Timeout: httpTimeout, CheckRedirect: checkReleaseRedirect},
			verified: make(map[string]ManifestAsset),
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
	s.downloadMu.Lock()
	defer s.downloadMu.Unlock()
	info, asset, err := s.resolveUpdate(ctx)
	if err != nil {
		return UpdateInfo{}, err
	}
	if !info.HasUpdate {
		info.Message = "当前已是最新版本"
		return info, nil
	}

	targetPath := updateAssetPath(info.LatestVersion, asset.AssetName)
	if err := os.MkdirAll(filepath.Dir(targetPath), 0o700); err != nil {
		return UpdateInfo{}, fmt.Errorf("failed to create update directory: %w", err)
	}

	if existingPath, err := verifyReleaseAsset(targetPath, asset); err == nil && existingPath != "" {
		info.DownloadedPath = existingPath
		s.remember(existingPath, asset)
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

	file, err := os.CreateTemp(filepath.Dir(targetPath), ".download-*")
	if err != nil {
		return UpdateInfo{}, fmt.Errorf("failed to create temp asset file: %w", err)
	}

	tempPath := file.Name()
	defer os.Remove(tempPath)
	hasher := sha256.New()
	writer := io.MultiWriter(file, hasher)
	written, copyErr := io.Copy(writer, io.LimitReader(resp.Body, asset.Size+1))
	if copyErr != nil {
		err := copyErr
		_ = file.Close()
		_ = os.Remove(tempPath)
		return UpdateInfo{}, fmt.Errorf("failed to write release asset: %w", err)
	}
	if err := file.Close(); err != nil {
		_ = os.Remove(tempPath)
		return UpdateInfo{}, fmt.Errorf("failed to close release asset: %w", err)
	}

	if written != asset.Size {
		return UpdateInfo{}, fmt.Errorf("release asset size mismatch")
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
	s.remember(targetPath, asset)
	info.RequiresManualInstall = requiresManualInstall(asset.InstallerType)
	info.Message = "更新包下载完成"
	return info, nil
}

func (s *Service) Install(downloadedPath string) (InstallResult, error) {
	s.downloadMu.Lock()
	defer s.downloadMu.Unlock()
	s.mu.Lock()
	asset, ok := s.verified[downloadedPath]
	s.mu.Unlock()
	if !ok {
		return InstallResult{}, fmt.Errorf("update must be verified through Check or Download before installation")
	}
	if err := validateAsset(asset); err != nil {
		return InstallResult{}, err
	}
	if _, err := verifyReleaseAsset(downloadedPath, asset); err != nil {
		return InstallResult{}, err
	}
	// Install a private verified snapshot, so changes to the original cached package
	// after verification cannot become the bytes executed or extracted.
	dir, err := os.MkdirTemp(filepath.Dir(downloadedPath), ".install-*")
	if err != nil {
		return InstallResult{}, err
	}
	sealed := filepath.Join(dir, asset.AssetName)
	src, err := os.Open(downloadedPath)
	if err != nil {
		os.RemoveAll(dir)
		return InstallResult{}, err
	}
	dst, err := os.OpenFile(sealed, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		src.Close()
		os.RemoveAll(dir)
		return InstallResult{}, err
	}
	written, copyErr := io.Copy(dst, io.LimitReader(src, asset.Size+1))
	src.Close()
	closeErr := dst.Close()
	if copyErr != nil || closeErr != nil || written != asset.Size {
		os.RemoveAll(dir)
		return InstallResult{}, fmt.Errorf("failed to prepare complete update snapshot")
	}
	if _, err := verifyReleaseAsset(sealed, asset); err != nil {
		os.RemoveAll(dir)
		return InstallResult{}, err
	}
	result, err := installUpdateAsset(sealed)
	if err != nil {
		os.RemoveAll(dir)
		return InstallResult{}, err
	}
	if result.OpenPath == "" {
		result.OpenPath = sealed
	}
	return result, nil
}
func (s *Service) remember(path string, asset ManifestAsset) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.verified == nil {
		s.verified = make(map[string]ManifestAsset)
	}
	s.verified[path] = asset
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

	if err := validateAsset(asset); err != nil {
		return UpdateInfo{}, ManifestAsset{}, err
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
	if existingPath, err := verifyReleaseAsset(downloadedPath, asset); err == nil && existingPath != "" {
		info.DownloadedPath = existingPath
		s.remember(existingPath, asset)
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
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&manifest); err != nil {
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
	if len(expectedSHA) != 64 {
		return "", fmt.Errorf("a SHA256 checksum is required")
	}
	if _, err := hex.DecodeString(expectedSHA); err != nil {
		return "", err
	}
	stat, err := os.Lstat(path)
	if err != nil {
		return "", err
	}
	if !stat.Mode().IsRegular() || stat.Size() > 512<<20 {
		return "", fmt.Errorf("update asset path %s is a directory", path)
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

func validateAsset(asset ManifestAsset) error {
	if asset.AssetName == "" || asset.AssetName == "." || asset.AssetName == ".." || strings.ContainsAny(asset.AssetName, "/\\:") || filepath.Base(asset.AssetName) != asset.AssetName {
		return fmt.Errorf("invalid release asset name")
	}
	if asset.Size <= 0 || asset.Size > 512<<20 {
		return fmt.Errorf("invalid release asset size")
	}
	if len(asset.SHA256) != 64 {
		return fmt.Errorf("release asset requires SHA256")
	}
	if _, err := hex.DecodeString(asset.SHA256); err != nil {
		return err
	}
	address, err := url.Parse(asset.URL)
	if err != nil || address.Scheme != "https" || address.Host != "github.com" || address.User != nil || !strings.HasPrefix(address.Path, "/"+repositorySlug+"/releases/download/") {
		return fmt.Errorf("release asset URL must belong to this GitHub repository")
	}
	switch runtimepkg.GOOS {
	case "darwin":
		if asset.OS != "darwin" || asset.Arch != "universal" || asset.InstallerType != "app-zip" || !strings.HasSuffix(strings.ToLower(asset.AssetName), ".zip") {
			return fmt.Errorf("release asset does not match macOS")
		}
	case "windows":
		if asset.OS != "windows" || asset.Arch != "amd64" || asset.InstallerType != "nsis" || !strings.HasSuffix(strings.ToLower(asset.AssetName), ".exe") {
			return fmt.Errorf("release asset does not match Windows")
		}
	default:
		return fmt.Errorf("unsupported update platform")
	}
	return nil
}
func checkReleaseRedirect(req *http.Request, via []*http.Request) error {
	if len(via) >= 10 {
		return fmt.Errorf("too many release redirects")
	}
	host := req.URL.Hostname()
	if req.URL.Scheme != "https" || req.URL.User != nil || (host != "github.com" && host != "release-assets.githubusercontent.com" && host != "objects.githubusercontent.com") {
		return fmt.Errorf("untrusted release redirect")
	}
	return nil
}

func verifyReleaseAsset(path string, asset ManifestAsset) (string, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return "", err
	}
	if info.Size() != asset.Size {
		return "", fmt.Errorf("cached update size does not match release manifest")
	}
	return verifyExistingAsset(path, asset.SHA256)
}
