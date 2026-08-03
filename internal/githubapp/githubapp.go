// Package githubapp authenticates as a GitHub App installation using the App's
// client id and private key, minting short-lived installation tokens used for
// both the commit-status API and git clones.
package githubapp

import (
	"context"
	"crypto/rsa"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/go-github/v89/github"

	"github.com/home-operations/downflate/internal/config"
)

// App authenticates as a GitHub App installation.
type App struct {
	install *installTransport
}

// New parses the App credentials and prepares the installation transport. It
// makes no network calls; the installation is discovered lazily on first use.
func New(cfg *config.Config) (*App, error) {
	key, err := jwt.ParseRSAPrivateKeyFromPEM(cfg.GitHubAppPrivateKey)
	if err != nil {
		return nil, fmt.Errorf("github app: parse private key: %w", err)
	}

	// App-level (JWT) client used to discover the installation and mint tokens.
	apps, err := github.NewClient(append(
		[]github.ClientOptionsFunc{github.WithTransport(&jwtTransport{
			clientID: cfg.GitHubAppClientID,
			key:      key,
		})},
		enterpriseOpts(cfg.Host)...,
	)...)
	if err != nil {
		return nil, fmt.Errorf("github app: client: %w", err)
	}

	return &App{install: &installTransport{apps: apps, owner: cfg.Owner, repo: cfg.Repo}}, nil
}

// HTTPClient returns an http.Client that authenticates API calls as the
// installation, for use with go-github.
func (a *App) HTTPClient() *http.Client {
	return &http.Client{Transport: a.install}
}

// Token returns a currently-valid installation access token, refreshing as
// needed. Usable as a git HTTPS password (username "x-access-token").
func (a *App) Token(ctx context.Context) (string, error) {
	return a.install.token(ctx)
}

// jwtTransport authenticates app-level API calls with a short-lived JWT signed
// by the App's private key, using the client id as the issuer (iss).
type jwtTransport struct {
	clientID string
	key      *rsa.PrivateKey
}

func (t *jwtTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	now := time.Now()
	signed, err := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.RegisteredClaims{
		Issuer:    t.clientID,
		IssuedAt:  jwt.NewNumericDate(now.Add(-30 * time.Second)),
		ExpiresAt: jwt.NewNumericDate(now.Add(9 * time.Minute)),
	}).SignedString(t.key)
	if err != nil {
		return nil, fmt.Errorf("github app: sign jwt: %w", err)
	}
	r := req.Clone(req.Context())
	r.Header.Set("Authorization", "Bearer "+signed)
	return http.DefaultTransport.RoundTrip(r)
}

// installTransport injects a cached, auto-refreshed installation access token
// into each request and forwards it to GitHub.
type installTransport struct {
	apps  *github.Client // app-JWT client used to discover the installation + mint tokens
	owner string
	repo  string

	mu     sync.Mutex
	instID int64
	tok    string
	exp    time.Time
}

func (t *installTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	tok, err := t.token(req.Context())
	if err != nil {
		return nil, err
	}
	r := req.Clone(req.Context())
	r.Header.Set("Authorization", "token "+tok)
	return http.DefaultTransport.RoundTrip(r)
}

// token returns a valid installation token, discovering the installation on
// first use and re-minting when the current token is within a minute of expiry.
func (t *installTransport) token(ctx context.Context) (string, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.tok != "" && time.Until(t.exp) > time.Minute {
		return t.tok, nil
	}
	if t.instID == 0 {
		inst, _, err := t.apps.Apps.GetRepositoryInstallation(ctx, t.owner, t.repo)
		if err != nil {
			return "", fmt.Errorf("github app: find installation for %s/%s: %w", t.owner, t.repo, err)
		}
		t.instID = inst.GetID()
	}
	it, _, err := t.apps.Apps.CreateInstallationToken(ctx, t.instID, nil)
	if err != nil {
		return "", fmt.Errorf("github app: mint installation token: %w", err)
	}
	t.tok, t.exp = it.GetToken(), it.GetExpiresAt().Time
	return t.tok, nil
}

// enterpriseOpts configures go-github for a GHES host, or returns nil for the
// public github.com.
func enterpriseOpts(host string) []github.ClientOptionsFunc {
	if host == "" || host == "github.com" {
		return nil
	}
	return []github.ClientOptionsFunc{github.WithEnterpriseURLs(
		fmt.Sprintf("https://%s/api/v3/", host),
		fmt.Sprintf("https://%s/api/uploads/", host),
	)}
}
