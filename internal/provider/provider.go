// Package provider publishes commit statuses back to the originating forge.
package provider

import (
	"context"
	"fmt"
	"net/http"

	"github.com/home-operations/downflate/internal/config"
)

// State is downflate's forge-neutral commit-status state.
type State string

// Commit-status states downflate reports.
const (
	Pending State = "pending"
	Success State = "success"
	Failure State = "failure"
)

// Status is a commit status to publish on a PR head commit.
type Status struct {
	SHA         string
	State       State
	Description string
	Context     string
	TargetURL   string
}

// Writer publishes a commit status to a forge.
type Writer interface {
	SetStatus(ctx context.Context, st Status) error
}

// New returns the Writer implementation for the configured forge. githubAppClient,
// when non-nil, is a pre-authenticated GitHub App installation client used
// instead of the PAT (ignored by non-GitHub forges).
func New(cfg *config.Config, githubAppClient *http.Client) (Writer, error) {
	switch cfg.Forge {
	case config.ForgeGitHub:
		return newGitHub(cfg, githubAppClient)
	case config.ForgeGitLab:
		return newGitLab(cfg)
	case config.ForgeForgejo:
		return newForgejo(cfg)
	default:
		return nil, fmt.Errorf("provider: unsupported forge %q", cfg.Forge)
	}
}

// truncate clamps a status description to GitHub's 140-character limit,
// respecting UTF-8 rune boundaries.
func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	if n <= 1 {
		return string(r[:n])
	}
	return string(r[:n-1]) + "…"
}
