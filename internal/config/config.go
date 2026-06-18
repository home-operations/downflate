// Package config loads downflate's runtime configuration from the environment.
package config

import (
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

// Forge identifies the git host downflate talks to.
type Forge string

// Supported forges.
const (
	ForgeGitHub  Forge = "github"
	ForgeGitLab  Forge = "gitlab"
	ForgeForgejo Forge = "forgejo"
)

// Config is the fully-resolved configuration for a downflate process.
type Config struct {
	Addr string // listen address, e.g. ":8080"

	Forge Forge  // github | gitlab | forgejo
	Host  string // forge host (empty ⇒ the forge's public host)
	Owner string // repo owner / namespace (all path segments but the last)
	Repo  string // repository name (last path segment)
	Path  string // full project path "owner/repo" or "group/sub/project" (GitLab pid)

	Token         string // forge API token (commit-status write)
	WebhookSecret string // HMAC secret; empty disables /hooks

	// GitHub App auth (GitHub only, optional). When the App's client id and
	// private key are set, downflate mints short-lived installation tokens for
	// both the status API and git clone instead of using Token. The installation
	// is resolved automatically from the repo.
	GitHubAppClientID   string
	GitHubAppPrivateKey []byte

	ClusterPath string // path inside the repo flate scans (e.g. "kubernetes")
	CacheDir    string // flate on-disk source cache (empty ⇒ flate default)

	Talosconfig  string   // path to talosconfig (mTLS creds + endpoints)
	TalosContext string   // talosconfig context name (empty ⇒ current)
	Nodes        []string // nodes to pull onto (empty ⇒ talosconfig nodes/endpoints)
	Namespace    string   // "cri" (default) | "system"

	GitDepth       int           // shallow clone depth (0 ⇒ full history)
	Timeout        time.Duration // per-PR render+pull deadline
	Concurrency    int           // max PRs processed in parallel
	RestrictEgress bool          // flate SSRF guard (untrusted/fork PRs)
	StatusContext  string        // commit-status context label

	LogLevel  slog.Level
	LogFormat string // "text" | "json"
}

// Load reads DOWNFLATE_* environment variables into a Config, applying
// defaults and validating required fields.
func Load() (*Config, error) {
	c := &Config{
		Addr:          envOr("DOWNFLATE_ADDR", ":8080"),
		Token:         os.Getenv("DOWNFLATE_TOKEN"),
		WebhookSecret: os.Getenv("DOWNFLATE_WEBHOOK_SECRET"),
		ClusterPath:   os.Getenv("DOWNFLATE_CLUSTER_PATH"),
		CacheDir:      os.Getenv("DOWNFLATE_CACHE_DIR"),
		Talosconfig:   envOr("DOWNFLATE_TALOSCONFIG", "/var/run/secrets/talos.dev/config"),
		TalosContext:  os.Getenv("DOWNFLATE_TALOS_CONTEXT"),
		Namespace:     strings.ToLower(envOr("DOWNFLATE_IMAGE_NAMESPACE", "cri")),
		StatusContext: envOr("DOWNFLATE_STATUS_CONTEXT", "downflate"),
		LogFormat:     strings.ToLower(envOr("DOWNFLATE_LOG_FORMAT", "text")),
	}

	if nodes := os.Getenv("DOWNFLATE_NODES"); nodes != "" {
		for n := range strings.SplitSeq(nodes, ",") {
			if n = strings.TrimSpace(n); n != "" {
				c.Nodes = append(c.Nodes, n)
			}
		}
	}

	c.GitHubAppClientID = os.Getenv("DOWNFLATE_GITHUB_APP_CLIENT_ID")

	var err error
	if err = c.loadAppPrivateKey(os.Getenv("DOWNFLATE_GITHUB_APP_PRIVATE_KEY")); err != nil {
		return nil, err
	}
	if c.GitDepth, err = intOr("DOWNFLATE_GIT_DEPTH", 1); err != nil {
		return nil, err
	}
	if c.Concurrency, err = intOr("DOWNFLATE_CONCURRENCY", 2); err != nil {
		return nil, err
	}
	if c.Concurrency < 1 {
		c.Concurrency = 1
	}
	if c.Timeout, err = durOr("DOWNFLATE_TIMEOUT", 15*time.Minute); err != nil {
		return nil, err
	}
	c.RestrictEgress = boolEnv("DOWNFLATE_RESTRICT_EGRESS")
	c.LogLevel = parseLevel(os.Getenv("DOWNFLATE_LOG_LEVEL"))

	if err := c.parseRepo(os.Getenv("DOWNFLATE_REPO")); err != nil {
		return nil, err
	}
	if c.Namespace != "cri" && c.Namespace != "system" {
		return nil, fmt.Errorf("DOWNFLATE_IMAGE_NAMESPACE must be \"cri\" or \"system\", got %q", c.Namespace)
	}
	// A GitHub App credential needs both halves, is GitHub-only, and makes Token
	// optional.
	if (c.GitHubAppClientID == "") != (len(c.GitHubAppPrivateKey) == 0) {
		return nil, fmt.Errorf("GitHub App auth needs both DOWNFLATE_GITHUB_APP_CLIENT_ID and DOWNFLATE_GITHUB_APP_PRIVATE_KEY (only one is set)")
	}
	if c.GitHubAppConfigured() && c.Forge != ForgeGitHub {
		return nil, fmt.Errorf("GitHub App auth is only valid for a github:// repo")
	}
	if c.Token == "" && !c.GitHubAppConfigured() {
		return nil, fmt.Errorf("DOWNFLATE_TOKEN or GitHub App credentials (DOWNFLATE_GITHUB_APP_CLIENT_ID + DOWNFLATE_GITHUB_APP_PRIVATE_KEY) are required")
	}
	return c, nil
}

// GitHubAppConfigured reports whether GitHub App credentials are present.
func (c *Config) GitHubAppConfigured() bool {
	return c.GitHubAppClientID != "" && len(c.GitHubAppPrivateKey) > 0
}

// loadAppPrivateKey reads the App private key from an inline PEM value or, when
// the value begins with "@", from the referenced file path.
func (c *Config) loadAppPrivateKey(v string) error {
	if v == "" {
		return nil
	}
	if path, ok := strings.CutPrefix(v, "@"); ok {
		b, err := os.ReadFile(path) //nolint:gosec // path is operator-supplied configuration

		if err != nil {
			return fmt.Errorf("DOWNFLATE_GITHUB_APP_PRIVATE_KEY: %w", err)
		}
		c.GitHubAppPrivateKey = b
		return nil
	}
	c.GitHubAppPrivateKey = []byte(v)
	return nil
}

// parseRepo decodes a forge URI like github://owner/repo,
// gitlab://gitlab.example.com/group/sub/project, or forgejo://git.example.com/owner/repo.
func (c *Config) parseRepo(raw string) error {
	if raw == "" {
		return fmt.Errorf("DOWNFLATE_REPO is required (e.g. github://owner/repo)")
	}
	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("DOWNFLATE_REPO: %w", err)
	}
	switch Forge(u.Scheme) {
	case ForgeGitHub, ForgeGitLab, ForgeForgejo:
		c.Forge = Forge(u.Scheme)
	default:
		return fmt.Errorf("DOWNFLATE_REPO: unknown forge scheme %q (want github|gitlab|forgejo)", u.Scheme)
	}

	// url.Parse puts the authority in u.Host, so for "github://owner/repo" the
	// owner lands in u.Host. Rejoin authority+path and re-split; the first
	// segment is the forge host only when it looks like one (contains a "." or
	// a port ":"), otherwise the host is implicit and it's the owner.
	parts := strings.Split(strings.Trim(u.Host+u.Path, "/"), "/")
	if len(parts) > 0 && (strings.Contains(parts[0], ".") || strings.Contains(parts[0], ":")) {
		c.Host = parts[0]
		parts = parts[1:]
	}
	if len(parts) < 2 {
		return fmt.Errorf("DOWNFLATE_REPO: path must be owner/repo, got %q", raw)
	}
	c.Path = strings.Join(parts, "/")
	c.Owner = strings.Join(parts[:len(parts)-1], "/")
	c.Repo = parts[len(parts)-1]
	return nil
}

// CloneURL builds the https clone URL for the configured repository.
func (c *Config) CloneURL() string {
	host := c.Host
	if host == "" {
		host = defaultHost(c.Forge)
	}
	return fmt.Sprintf("https://%s/%s.git", host, c.Path)
}

func defaultHost(f Forge) string {
	switch f {
	case ForgeGitLab:
		return "gitlab.com"
	default:
		return "github.com"
	}
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func boolEnv(key string) bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv(key)))
	return v == "1" || v == "true" || v == "yes"
}

func intOr(key string, def int) (int, error) {
	v := os.Getenv(key)
	if v == "" {
		return def, nil
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", key, err)
	}
	return n, nil
}

func durOr(key string, def time.Duration) (time.Duration, error) {
	v := os.Getenv(key)
	if v == "" {
		return def, nil
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", key, err)
	}
	return d, nil
}

func parseLevel(s string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
