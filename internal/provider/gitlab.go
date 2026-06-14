package provider

import (
	"context"
	"fmt"

	gitlab "gitlab.com/gitlab-org/api/client-go"

	"github.com/home-operations/downflate/internal/config"
)

type gitlabWriter struct {
	c       *gitlab.Client
	project string // numeric ID or URL-path "group/sub/project"
}

func newGitLab(cfg *config.Config) (Writer, error) {
	var opts []gitlab.ClientOptionFunc
	if cfg.Host != "" && cfg.Host != "gitlab.com" {
		opts = append(opts, gitlab.WithBaseURL(fmt.Sprintf("https://%s/api/v4", cfg.Host)))
	}
	c, err := gitlab.NewClient(cfg.Token, opts...)
	if err != nil {
		return nil, fmt.Errorf("gitlab client: %w", err)
	}
	return &gitlabWriter{c: c, project: cfg.Path}, nil
}

func (w *gitlabWriter) SetStatus(ctx context.Context, st Status) error {
	_, _, err := w.c.Commits.SetCommitStatus(w.project, st.SHA, &gitlab.SetCommitStatusOptions{
		State:       gitlabState(st.State),
		Context:     new(st.Context),
		Description: new(truncate(st.Description, 140)),
		TargetURL:   nilIfEmpty(st.TargetURL),
	}, gitlab.WithContext(ctx))
	return err
}

func gitlabState(s State) gitlab.BuildStateValue {
	switch s {
	case Pending:
		return gitlab.Pending
	case Success:
		return gitlab.Success
	default:
		return gitlab.Failed
	}
}
