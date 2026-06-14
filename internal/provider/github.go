package provider

import (
	"context"
	"fmt"
	"net/http"

	"github.com/google/go-github/v88/github"

	"github.com/home-operations/downflate/internal/config"
)

type githubWriter struct {
	c     *github.Client
	owner string
	repo  string
}

func newGitHub(cfg *config.Config, appClient *http.Client) (Writer, error) {
	var opts []github.ClientOptionsFunc
	if appClient != nil {
		opts = append(opts, github.WithHTTPClient(appClient))
	} else {
		opts = append(opts, github.WithAuthToken(cfg.Token))
	}
	if cfg.Host != "" && cfg.Host != "github.com" {
		opts = append(opts, github.WithEnterpriseURLs(
			fmt.Sprintf("https://%s/api/v3/", cfg.Host),
			fmt.Sprintf("https://%s/api/uploads/", cfg.Host),
		))
	}
	c, err := github.NewClient(opts...)
	if err != nil {
		return nil, fmt.Errorf("github client: %w", err)
	}
	owner, repo, err := splitOwnerRepo(cfg.Path)
	if err != nil {
		return nil, err
	}
	return &githubWriter{c: c, owner: owner, repo: repo}, nil
}

func (w *githubWriter) SetStatus(ctx context.Context, st Status) error {
	_, _, err := w.c.Repositories.CreateStatus(ctx, w.owner, w.repo, st.SHA, github.RepoStatus{
		State:       new(string(st.State)), // pending|success|failure are valid GitHub states
		Description: new(truncate(st.Description, 140)),
		Context:     new(st.Context),
		TargetURL:   nilIfEmpty(st.TargetURL),
	})
	return err
}

func splitOwnerRepo(path string) (owner, repo string, err error) {
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' {
			return path[:i], path[i+1:], nil
		}
	}
	return "", "", fmt.Errorf("provider: invalid repo path %q", path)
}

func nilIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
