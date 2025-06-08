package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

// GitHubClient handles GitHub API interactions for update checking
type GitHubClient struct {
	repository  string
	token       string // Optional for higher rate limits
	httpClient  *http.Client
	rateLimiter *RateLimiter
	mutex       sync.RWMutex
	lastCheck   time.Time
	cacheExpiry time.Duration
	cachedData  map[string]*cachedResponse
}

// RateLimiter manages GitHub API rate limiting
type RateLimiter struct {
	requests     int
	resetTime    time.Time
	remaining    int
	limit        int
	mutex        sync.RWMutex
}

// cachedResponse stores cached API responses
type cachedResponse struct {
	data      interface{}
	timestamp time.Time
}

// Release represents a GitHub release
type Release struct {
	TagName         string    `json:"tag_name"`
	Name            string    `json:"name"`
	Body            string    `json:"body"`
	Prerelease      bool      `json:"prerelease"`
	Draft           bool      `json:"draft"`
	Assets          []Asset   `json:"assets"`
	PublishedAt     time.Time `json:"published_at"`
	HTMLURL         string    `json:"html_url"`
	TarballURL      string    `json:"tarball_url"`
	ZipballURL      string    `json:"zipball_url"`
	ID              int64     `json:"id"`
	NodeID          string    `json:"node_id"`
	TargetCommitish string    `json:"target_commitish"`
}

// Asset represents a release asset
type Asset struct {
	ID                 int64     `json:"id"`
	NodeID             string    `json:"node_id"`
	Name               string    `json:"name"`
	Label              string    `json:"label"`
	ContentType        string    `json:"content_type"`
	State              string    `json:"state"`
	Size               int64     `json:"size"`
	DownloadCount      int       `json:"download_count"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
	BrowserDownloadURL string    `json:"browser_download_url"`
	Uploader           User      `json:"uploader"`
}

// User represents a GitHub user
type User struct {
	Login     string `json:"login"`
	ID        int64  `json:"id"`
	NodeID    string `json:"node_id"`
	AvatarURL string `json:"avatar_url"`
	HTMLURL   string `json:"html_url"`
	Type      string `json:"type"`
}

// UpdateInfo contains information about available updates
type UpdateInfo struct {
	HasUpdate       bool
	CurrentVersion  string
	LatestVersion   string
	LatestRelease   *Release
	ReleaseNotes    string
	IsPrerelease    bool
	PublishedAt     time.Time
	DownloadURL     string
	AssetName       string
	AssetSize       int64
}

// NewGitHubClient creates a new GitHub API client
func NewGitHubClient(repository string, token string) *GitHubClient {
	return &GitHubClient{
		repository: repository,
		token:      token,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		rateLimiter: &RateLimiter{
			limit:     60, // Default rate limit for unauthenticated requests
			remaining: 60,
			resetTime: time.Now().Add(time.Hour),
		},
		cacheExpiry: 10 * time.Minute, // Cache responses for 10 minutes
		cachedData:  make(map[string]*cachedResponse),
	}
}

// SetToken sets the GitHub API token for authenticated requests
func (gc *GitHubClient) SetToken(token string) {
	gc.mutex.Lock()
	defer gc.mutex.Unlock()
	gc.token = token
	// Authenticated requests have higher rate limits
	gc.rateLimiter.limit = 5000
	gc.rateLimiter.remaining = 5000
}

// GetLatestRelease fetches the latest release for the specified channel
func (gc *GitHubClient) GetLatestRelease(channel string) (*Release, error) {
	releases, err := gc.GetReleases(10) // Get last 10 releases
	if err != nil {
		return nil, err
	}

	for _, release := range releases {
		// Skip drafts
		if release.Draft {
			continue
		}

		version := GetVersionFromTag(release.TagName)
		if MatchesChannel(version, channel) {
			return release, nil
		}
	}

	return nil, fmt.Errorf("no release found for channel: %s", channel)
}

// GetReleases fetches releases from GitHub API
func (gc *GitHubClient) GetReleases(limit int) ([]*Release, error) {
	if limit <= 0 || limit > 100 {
		limit = 30 // Default limit
	}

	// Check cache first
	cacheKey := fmt.Sprintf("releases_%d", limit)
	if cached := gc.getCached(cacheKey); cached != nil {
		if releases, ok := cached.([]*Release); ok {
			return releases, nil
		}
	}

	// Check rate limiting
	if err := gc.checkRateLimit(); err != nil {
		return nil, err
	}

	url := fmt.Sprintf("https://api.github.com/repos/%s/releases?per_page=%d", gc.repository, limit)
	
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	// Add authentication if token is available
	if gc.token != "" {
		req.Header.Set("Authorization", "Bearer "+gc.token)
	}

	// Set user agent
	req.Header.Set("User-Agent", "Delta-CLI/"+GetVersionShort())
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := gc.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch releases: %v", err)
	}
	defer resp.Body.Close()

	// Update rate limiting info
	gc.updateRateLimit(resp)

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GitHub API error: %d - %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %v", err)
	}

	var releases []*Release
	if err := json.Unmarshal(body, &releases); err != nil {
		return nil, fmt.Errorf("failed to parse releases: %v", err)
	}

	// Cache the response
	gc.setCached(cacheKey, releases)

	return releases, nil
}

// GetReleaseByTag fetches a specific release by tag
func (gc *GitHubClient) GetReleaseByTag(tag string) (*Release, error) {
	// Check cache first
	cacheKey := fmt.Sprintf("release_%s", tag)
	if cached := gc.getCached(cacheKey); cached != nil {
		if release, ok := cached.(*Release); ok {
			return release, nil
		}
	}

	// Check rate limiting
	if err := gc.checkRateLimit(); err != nil {
		return nil, err
	}

	url := fmt.Sprintf("https://api.github.com/repos/%s/releases/tags/%s", gc.repository, tag)
	
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	// Add authentication if token is available
	if gc.token != "" {
		req.Header.Set("Authorization", "Bearer "+gc.token)
	}

	req.Header.Set("User-Agent", "Delta-CLI/"+GetVersionShort())
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := gc.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch release: %v", err)
	}
	defer resp.Body.Close()

	// Update rate limiting info
	gc.updateRateLimit(resp)

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("release not found: %s", tag)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GitHub API error: %d - %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %v", err)
	}

	var release Release
	if err := json.Unmarshal(body, &release); err != nil {
		return nil, fmt.Errorf("failed to parse release: %v", err)
	}

	// Cache the response
	gc.setCached(cacheKey, &release)

	return &release, nil
}

// GetAssetDownloadURL returns the download URL for a specific asset
func (gc *GitHubClient) GetAssetDownloadURL(asset *Asset) (string, error) {
	if asset == nil {
		return "", fmt.Errorf("asset is nil")
	}
	return asset.BrowserDownloadURL, nil
}

// SelectAssetForPlatform selects the appropriate asset for the current platform
func (gc *GitHubClient) SelectAssetForPlatform(assets []Asset) (*Asset, error) {
	if len(assets) == 0 {
		return nil, fmt.Errorf("no assets available")
	}

	// Platform-specific patterns
	platformPatterns := map[string][]string{
		"linux":   {"linux", "Linux"},
		"darwin":  {"darwin", "macos", "mac", "Darwin", "macOS"},
		"windows": {"windows", "win", "Windows", "Win"},
	}

	// Architecture patterns
	archPatterns := map[string][]string{
		"amd64": {"amd64", "x86_64", "x64"},
		"arm64": {"arm64", "aarch64"},
		"386":   {"386", "i386", "x86"},
	}

	currentOS := getCurrentOS()
	currentArch := getCurrentArch()

	// First, try to find exact platform and architecture match
	for _, asset := range assets {
		name := strings.ToLower(asset.Name)
		
		// Check if this asset matches our platform
		platformMatch := false
		for _, pattern := range platformPatterns[currentOS] {
			if strings.Contains(name, strings.ToLower(pattern)) {
				platformMatch = true
				break
			}
		}

		if !platformMatch {
			continue
		}

		// Check if this asset matches our architecture
		archMatch := false
		for _, pattern := range archPatterns[currentArch] {
			if strings.Contains(name, strings.ToLower(pattern)) {
				archMatch = true
				break
			}
		}

		if archMatch {
			return &asset, nil
		}
	}

	// Fallback: try to find just platform match
	for _, asset := range assets {
		name := strings.ToLower(asset.Name)
		
		for _, pattern := range platformPatterns[currentOS] {
			if strings.Contains(name, strings.ToLower(pattern)) {
				return &asset, nil
			}
		}
	}

	// Last resort: return the first asset
	return &assets[0], nil
}

// checkRateLimit checks if we're within rate limits
func (gc *GitHubClient) checkRateLimit() error {
	gc.rateLimiter.mutex.RLock()
	defer gc.rateLimiter.mutex.RUnlock()

	now := time.Now()
	if now.After(gc.rateLimiter.resetTime) {
		// Rate limit has reset
		gc.rateLimiter.remaining = gc.rateLimiter.limit
		gc.rateLimiter.resetTime = now.Add(time.Hour)
		return nil
	}

	if gc.rateLimiter.remaining <= 0 {
		waitTime := gc.rateLimiter.resetTime.Sub(now)
		return fmt.Errorf("rate limit exceeded, wait %v before next request", waitTime)
	}

	return nil
}

// updateRateLimit updates rate limiting information from response headers
func (gc *GitHubClient) updateRateLimit(resp *http.Response) {
	gc.rateLimiter.mutex.Lock()
	defer gc.rateLimiter.mutex.Unlock()

	// Decrement remaining requests
	gc.rateLimiter.remaining--

	// Update from headers if available
	if remaining := resp.Header.Get("X-RateLimit-Remaining"); remaining != "" {
		if val, err := parseInt(remaining); err == nil {
			gc.rateLimiter.remaining = val
		}
	}

	if reset := resp.Header.Get("X-RateLimit-Reset"); reset != "" {
		if val, err := parseInt(reset); err == nil {
			gc.rateLimiter.resetTime = time.Unix(int64(val), 0)
		}
	}
}

// getCached retrieves cached data if still valid
func (gc *GitHubClient) getCached(key string) interface{} {
	gc.mutex.RLock()
	defer gc.mutex.RUnlock()

	if cached, exists := gc.cachedData[key]; exists {
		if time.Since(cached.timestamp) < gc.cacheExpiry {
			return cached.data
		}
		// Remove expired cache
		delete(gc.cachedData, key)
	}

	return nil
}

// setCached stores data in cache
func (gc *GitHubClient) setCached(key string, data interface{}) {
	gc.mutex.Lock()
	defer gc.mutex.Unlock()

	gc.cachedData[key] = &cachedResponse{
		data:      data,
		timestamp: time.Now(),
	}
}

// GetRateLimitStatus returns current rate limit status
func (gc *GitHubClient) GetRateLimitStatus() map[string]interface{} {
	gc.rateLimiter.mutex.RLock()
	defer gc.rateLimiter.mutex.RUnlock()

	return map[string]interface{}{
		"limit":     gc.rateLimiter.limit,
		"remaining": gc.rateLimiter.remaining,
		"reset_time": gc.rateLimiter.resetTime,
		"wait_time": gc.rateLimiter.resetTime.Sub(time.Now()),
	}
}

// Helper functions
func parseInt(s string) (int, error) {
	return strconv.Atoi(s)
}

func getCurrentOS() string {
	return runtime.GOOS
}

func getCurrentArch() string {
	return runtime.GOARCH
}