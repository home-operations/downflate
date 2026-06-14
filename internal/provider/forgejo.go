package provider

import (
	"context"
	"fmt"

	forgejo "codeberg.org/mvdkleijn/forgejo-sdk/forgejo/v3"

	"github.com/home-operations/downflate/internal/config"
)

type forgejoWriter struct {
	c     *forgejo.Client
	owner string
	repo  string
}

func newForgejo(cfg *config.Config) (Writer, error) {
	host := cfg.Host
	if host == "" {
		return nil, fmt.Errorf("forgejo: DOWNFLATE_REPO must include a host (forgejo://host/owner/repo)")
	}
	c, err := forgejo.NewClient(fmt.Sprintf("https://%s", host), forgejo.SetToken(cfg.Token))
	if err != nil {
		return nil, fmt.Errorf("forgejo client: %w", err)
	}
	owner, repo, err := splitOwnerRepo(cfg.Path)
	if err != nil {
		return nil, err
	}
	return &forgejoWriter{c: c, owner: owner, repo: repo}, nil
}

func (w *forgejoWriter) SetStatus(_ context.Context, st Status) error {
	_, _, err := w.c.CreateStatus(w.owner, w.repo, st.SHA, forgejo.CreateStatusOption{
		State:       forgejoState(st.State),
		Description: truncate(st.Description, 140),
		Context:     st.Context,
		TargetURL:   st.TargetURL,
	})
	return err
}

func forgejoState(s State) forgejo.StatusState {
	switch s {
	case Pending:
		return forgejo.StatusPending
	case Success:
		return forgejo.StatusSuccess
	default:
		return forgejo.StatusFailure
	}
}
