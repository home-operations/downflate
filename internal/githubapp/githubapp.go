// Package githubapp provides GitHub App installation authentication, minting
// short-lived tokens used for both the commit-status API and git clones.
package githubapp

import (
	"context"
	"fmt"
	"net/http"

	"github.com/bradleyfalzon/ghinstallation/v2"
	"github.com/google/go-github/v88/github"

	"github.com/home-operations/downflate/internal/config"
)

// App authenticates as a GitHub App installation.
type App struct {
	itr *ghinstallation.Transport
}

// New builds an App from the configured credentials. When no installation ID is
// configured it is discovered from the repository.
func New(ctx context.Context, cfg *config.Config) (*App, error) {
	base := enterpriseBase(cfg.Host)

	atr, err := ghinstallation.NewAppsTransport(http.DefaultTransport, cfg.GitHubAppID, cfg.GitHubAppPrivateKey)
	if err != nil {
		return nil, fmt.Errorf("github app transport: %w", err)
	}
	if base != "" {
		atr.BaseURL = base
	}

	installID := cfg.GitHubAppInstallationID
	if installID == 0 {
		if installID, err = discover(ctx, atr, base, cfg.Owner, cfg.Repo); err != nil {
			return nil, err
		}
	}

	itr := ghinstallation.NewFromAppsTransport(atr, installID)
	if base != "" {
		itr.BaseURL = base
	}
	return &App{itr: itr}, nil
}

// HTTPClient returns an http.Client that authenticates API calls as the
// installation, for use with go-github.
func (a *App) HTTPClient() *http.Client {
	return &http.Client{Transport: a.itr}
}

// Token returns a currently-valid installation access token, refreshing it as
// needed. Suitable as a git HTTPS password (username "x-access-token").
func (a *App) Token(ctx context.Context) (string, error) {
	return a.itr.Token(ctx)
}

// discover resolves the installation ID for owner/repo using app-level (JWT) auth.
func discover(ctx context.Context, atr *ghinstallation.AppsTransport, base, owner, repo string) (int64, error) {
	opts := []github.ClientOptionsFunc{github.WithHTTPClient(&http.Client{Transport: atr})}
	if base != "" {
		opts = append(opts, github.WithEnterpriseURLs(base+"/", base+"/"))
	}
	c, err := github.NewClient(opts...)
	if err != nil {
		return 0, fmt.Errorf("github app discovery client: %w", err)
	}
	inst, _, err := c.Apps.GetRepositoryInstallation(ctx, owner, repo)
	if err != nil {
		return 0, fmt.Errorf("find installation for %s/%s: %w", owner, repo, err)
	}
	return inst.GetID(), nil
}

// enterpriseBase returns the GHES API base URL, or "" for public github.com.
func enterpriseBase(host string) string {
	if host == "" || host == "github.com" {
		return ""
	}
	return fmt.Sprintf("https://%s/api/v3", host)
}
