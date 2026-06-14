// Package render wraps flate's orchestrator to render the two sides of a PR and
// extract the container images the change introduces.
package render

import (
	"context"
	"path/filepath"

	"github.com/home-operations/flate/pkg/orchestrator"

	"github.com/home-operations/downflate/internal/config"
)

// Renderer renders PR diffs via flate. It is safe for concurrent use; flate
// guards its on-disk cache with file locks.
type Renderer struct {
	cfg *config.Config
}

// New returns a Renderer bound to cfg.
func New(cfg *config.Config) *Renderer {
	return &Renderer{cfg: cfg}
}

// Result is the changed-only render of both sides of a PR.
type Result struct {
	Base *orchestrator.Result
	Head *orchestrator.Result
	// Failures is the number of resources that failed to render across both
	// sides — surfaced into the commit-status description.
	Failures int
}

// Render renders baseDir and headDir in flate's changed-only mode. A non-nil
// error from flate is advisory (partial results may still be present); callers
// inspect Result.Base/Head and Failures.
func (r *Renderer) Render(ctx context.Context, baseDir, headDir string) (*Result, error) {
	cfg := orchestrator.Config{
		WipeSecrets:         true,
		AllowMissingSecrets: true,
		GitDepth:            r.cfg.GitDepth,
		RestrictEgress:      r.cfg.RestrictEgress,
		CacheDir:            r.cfg.CacheDir,
	}
	base, head, err := orchestrator.RenderTrees(ctx, r.tree(baseDir), r.tree(headDir), cfg)

	out := &Result{Base: base.Result, Head: head.Result}
	if base.Result != nil {
		out.Failures += len(base.Result.Failed)
	}
	if head.Result != nil {
		out.Failures += len(head.Result.Failed)
	}
	return out, err
}

func (r *Renderer) tree(dir string) orchestrator.Tree {
	t := orchestrator.Tree{RepoRoot: dir}
	if r.cfg.ClusterPath != "" {
		t.Path = filepath.Join(dir, r.cfg.ClusterPath)
	}
	return t
}
