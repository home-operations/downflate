// Package git materializes a PR's base and head trees on disk so flate can
// render both sides of the diff.
package git

import (
	"context"
	"fmt"
	"os"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/transport"
	githttp "github.com/go-git/go-git/v5/plumbing/transport/http"

	"github.com/home-operations/downflate/internal/config"
)

// Trees holds two checked-out working trees for one comparison.
type Trees struct {
	BaseDir string
	HeadDir string
	root    string
}

// Cleanup removes both working trees. Safe to call on a zero/partial Trees.
func (t *Trees) Cleanup() error {
	if t == nil || t.root == "" {
		return nil
	}
	return os.RemoveAll(t.root)
}

// Materialize shallow-clones the base branch and the PR head ref into two
// sibling directories, authenticating with the supplied token (a PAT or a fresh
// GitHub App installation token). The caller must invoke Trees.Cleanup.
func Materialize(ctx context.Context, cfg *config.Config, token, baseBranch, headRef string) (*Trees, error) {
	root, err := os.MkdirTemp("", "downflate-")
	if err != nil {
		return nil, fmt.Errorf("tempdir: %w", err)
	}
	t := &Trees{
		BaseDir: root + "/base",
		HeadDir: root + "/head",
		root:    root,
	}

	url := cfg.CloneURL()
	auth := authFor(cfg.Forge, token)

	if err := clone(ctx, t.BaseDir, url, auth, plumbing.NewBranchReferenceName(baseBranch), cfg.GitDepth); err != nil {
		_ = t.Cleanup()
		return nil, fmt.Errorf("clone base %q: %w", baseBranch, err)
	}
	if err := clone(ctx, t.HeadDir, url, auth, plumbing.ReferenceName(headRef), cfg.GitDepth); err != nil {
		_ = t.Cleanup()
		return nil, fmt.Errorf("clone head %q: %w", headRef, err)
	}
	return t, nil
}

func clone(ctx context.Context, dir, url string, auth transport.AuthMethod, ref plumbing.ReferenceName, depth int) error {
	_, err := gogit.PlainCloneContext(ctx, dir, false, &gogit.CloneOptions{
		URL:           url,
		Auth:          auth,
		ReferenceName: ref,
		SingleBranch:  true,
		Depth:         depth,
		Tags:          gogit.NoTags,
	})
	return err
}

// authFor builds basic-auth credentials appropriate to each forge's token model.
func authFor(forge config.Forge, token string) transport.AuthMethod {
	if token == "" {
		return nil
	}
	switch forge {
	case config.ForgeGitLab:
		return &githttp.BasicAuth{Username: "oauth2", Password: token}
	case config.ForgeForgejo:
		return &githttp.BasicAuth{Username: token, Password: token}
	default: // GitHub (PAT or App installation token)
		return &githttp.BasicAuth{Username: "x-access-token", Password: token}
	}
}
