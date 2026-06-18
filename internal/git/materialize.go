// Package git materializes a PR's base and head trees on disk so flate can
// render both sides of the diff.
package git

import (
	"context"
	"fmt"
	"os"

	gogit "github.com/go-git/go-git/v5"
	gitconfig "github.com/go-git/go-git/v5/config"
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

	if err := fetchTree(ctx, t.BaseDir, url, plumbing.NewBranchReferenceName(baseBranch).String(), auth, cfg.GitDepth); err != nil {
		_ = t.Cleanup()
		return nil, fmt.Errorf("fetch base %q: %w", baseBranch, err)
	}
	if err := fetchTree(ctx, t.HeadDir, url, headRef, auth, cfg.GitDepth); err != nil {
		_ = t.Cleanup()
		return nil, fmt.Errorf("fetch head %q: %w", headRef, err)
	}
	return t, nil
}

// localRef is the throwaway local branch each side is fetched into and checked
// out, so arbitrary source refs work uniformly. go-git's SingleBranch clone
// only handles refs under refs/heads/, which mangles non-branch refs like
// refs/pull/N/head; fetching into a local branch via an explicit refspec does not.
const localRef = "downflate"

// fetchTree initializes a repo, shallow-fetches a single source ref into a local
// branch, and checks it out into dir.
func fetchTree(ctx context.Context, dir, url, srcRef string, auth transport.AuthMethod, depth int) error {
	repo, err := gogit.PlainInit(dir, false)
	if err != nil {
		return fmt.Errorf("init: %w", err)
	}
	if _, err := repo.CreateRemote(&gitconfig.RemoteConfig{
		Name: "origin",
		URLs: []string{url},
	}); err != nil {
		return fmt.Errorf("remote: %w", err)
	}
	spec := gitconfig.RefSpec(fmt.Sprintf("+%s:refs/heads/%s", srcRef, localRef))
	if err := repo.FetchContext(ctx, &gogit.FetchOptions{
		RemoteName: "origin",
		Auth:       auth,
		RefSpecs:   []gitconfig.RefSpec{spec},
		Depth:      depth,
		Tags:       gogit.NoTags,
	}); err != nil {
		return fmt.Errorf("fetch %s: %w", srcRef, err)
	}
	wt, err := repo.Worktree()
	if err != nil {
		return fmt.Errorf("worktree: %w", err)
	}
	if err := wt.Checkout(&gogit.CheckoutOptions{Branch: plumbing.NewBranchReferenceName(localRef)}); err != nil {
		return fmt.Errorf("checkout: %w", err)
	}
	return nil
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
