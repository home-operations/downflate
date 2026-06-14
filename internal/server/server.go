// Package server wires the webhook intake to the render→pull→status pipeline.
package server

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/home-operations/downflate/internal/config"
	"github.com/home-operations/downflate/internal/git"
	"github.com/home-operations/downflate/internal/provider"
	"github.com/home-operations/downflate/internal/render"
	"github.com/home-operations/downflate/internal/talos"
	"github.com/home-operations/downflate/internal/webhook"
)

// Renderer renders both sides of a PR diff. *render.Renderer satisfies it.
type Renderer interface {
	Render(ctx context.Context, baseDir, headDir string) (*render.Result, error)
}

// Puller pre-pulls image references onto the cluster. *talos.Puller satisfies it.
type Puller interface {
	PullAll(ctx context.Context, refs []string) []talos.Result
	Nodes() []string
}

// Server processes pull-request webhooks: it renders the diff, pre-pulls the
// changed images onto the cluster, and reports a commit status.
type Server struct {
	cfg      *config.Config
	log      *slog.Logger
	writer   provider.Writer
	renderer Renderer
	puller   Puller
	baseCtx  context.Context

	// gitToken resolves the credential used for cloning (a PAT or a fresh
	// GitHub App installation token).
	gitToken func(ctx context.Context) (string, error)
	// materialize checks out the base and head trees; overridable in tests.
	materialize func(ctx context.Context, baseBranch, headRef string) (*git.Trees, error)

	mu      sync.Mutex
	running map[int]bool          // PRs currently being processed
	rerun   map[int]webhook.Event // latest event that arrived mid-flight, per PR
	sem     chan struct{}         // global concurrency limiter
	wg      sync.WaitGroup        // tracks in-flight jobs for graceful shutdown
}

// New constructs a Server. baseCtx bounds the lifetime of all background jobs.
// gitToken resolves the clone credential; nil falls back to cfg.Token (PAT).
func New(baseCtx context.Context, cfg *config.Config, w provider.Writer, r Renderer, p Puller, gitToken func(context.Context) (string, error)) *Server {
	if gitToken == nil {
		gitToken = func(context.Context) (string, error) { return cfg.Token, nil }
	}
	s := &Server{
		cfg:      cfg,
		log:      slog.Default(),
		writer:   w,
		renderer: r,
		puller:   p,
		baseCtx:  baseCtx,
		gitToken: gitToken,
		running:  make(map[int]bool),
		rerun:    make(map[int]webhook.Event),
		sem:      make(chan struct{}, cfg.Concurrency),
	}
	s.materialize = func(ctx context.Context, baseBranch, headRef string) (*git.Trees, error) {
		token, err := s.gitToken(ctx)
		if err != nil {
			return nil, fmt.Errorf("resolve git token: %w", err)
		}
		return git.Materialize(ctx, cfg, token, baseBranch, headRef)
	}
	return s
}

// submit schedules an event for processing, coalescing per PR: if a PR is
// already in flight, the newest event replaces any queued rerun and the run is
// retried once the current one finishes (latest-wins).
func (s *Server) submit(ev webhook.Event) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.running[ev.Number] {
		s.rerun[ev.Number] = ev
		return
	}
	s.running[ev.Number] = true
	s.dispatch(ev)
}

// dispatch starts a job goroutine. Caller holds s.mu.
func (s *Server) dispatch(ev webhook.Event) {
	s.wg.Go(func() {
		s.sem <- struct{}{}
		s.process(s.baseCtx, ev)
		<-s.sem

		s.mu.Lock()
		defer s.mu.Unlock()
		if next, ok := s.rerun[ev.Number]; ok {
			delete(s.rerun, ev.Number)
			s.dispatch(next)
			return
		}
		delete(s.running, ev.Number)
	})
}

// Shutdown blocks until all in-flight jobs finish.
func (s *Server) Shutdown() { s.wg.Wait() }

// process runs the full pipeline for one PR event.
func (s *Server) process(parent context.Context, ev webhook.Event) {
	ctx, cancel := context.WithTimeout(parent, s.cfg.Timeout)
	defer cancel()
	log := s.log.With("pr", ev.Number, "sha", short(ev.HeadSHA))

	s.setStatus(ctx, ev, provider.Pending, "rendering changed images")

	trees, err := s.materialize(ctx, ev.Base, ev.HeadRef(s.cfg.Forge))
	if err != nil {
		log.Error("materialize failed", "err", err)
		s.setStatus(ctx, ev, provider.Failure, "git clone failed")
		return
	}
	defer func() {
		if err := trees.Cleanup(); err != nil {
			log.Warn("cleanup failed", "err", err)
		}
	}()

	res, rerr := s.renderer.Render(ctx, trees.BaseDir, trees.HeadDir)
	if rerr != nil {
		log.Warn("render reported failures", "err", rerr, "failures", res.Failures)
	}

	images := render.Changed(res.Base, res.Head)
	nodes := len(s.puller.Nodes())
	if len(images) == 0 {
		log.Info("no changed images", "render_failures", res.Failures)
		s.setStatus(ctx, ev, provider.Success, statusDesc(res.Failures, 0, 0, nodes))
		return
	}

	refs := make([]string, len(images))
	for i, im := range images {
		refs[i] = im.Ref
	}
	log.Info("pre-pulling images", "images", len(images), "nodes", nodes)

	failed := 0
	for _, r := range s.puller.PullAll(ctx, refs) {
		if r.Err != nil {
			failed++
			log.Error("pull failed", "node", r.Node, "ref", r.Ref, "err", r.Err)
		}
	}

	state := provider.Success
	if failed > 0 {
		state = provider.Failure
	}
	s.setStatus(ctx, ev, state, statusDesc(res.Failures, len(images), failed, nodes))
	log.Info("done", "state", state, "images", len(images), "pull_failures", failed, "render_failures", res.Failures)
}

// setStatus publishes a commit status, using a detached short-lived context so
// the final status still lands even when the job's deadline has expired.
func (s *Server) setStatus(parent context.Context, ev webhook.Event, state provider.State, desc string) {
	ctx, cancel := context.WithTimeout(context.WithoutCancel(parent), 30*time.Second)
	defer cancel()
	st := provider.Status{
		SHA:         ev.HeadSHA,
		State:       state,
		Description: desc,
		Context:     s.cfg.StatusContext,
	}
	if err := s.writer.SetStatus(ctx, st); err != nil {
		s.log.Error("set status failed", "pr", ev.Number, "state", state, "err", err)
	}
}

func statusDesc(renderFailures, images, pullFailures, nodes int) string {
	switch {
	case images == 0:
		if renderFailures > 0 {
			return fmt.Sprintf("no image changes (%d render failures)", renderFailures)
		}
		return "no image changes"
	case pullFailures > 0:
		return fmt.Sprintf("%d/%d pulls failed across %d nodes", pullFailures, images*nodes, nodes)
	default:
		return fmt.Sprintf("pre-pulled %d images across %d nodes", images, nodes)
	}
}

func short(sha string) string {
	if len(sha) > 7 {
		return sha[:7]
	}
	return sha
}
