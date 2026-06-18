// Command downflate pre-pulls a pull request's changed container images onto a
// Talos Linux cluster and reports the result back as a commit status.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/home-operations/downflate/internal/config"
	"github.com/home-operations/downflate/internal/githubapp"
	"github.com/home-operations/downflate/internal/provider"
	"github.com/home-operations/downflate/internal/render"
	"github.com/home-operations/downflate/internal/server"
	"github.com/home-operations/downflate/internal/talos"
)

// Build metadata, injected via -ldflags at build time.
var (
	version = "dev"
	commit  = "dev"
)

func main() {
	if err := run(); err != nil {
		slog.Error("fatal", "err", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	setupLogging(cfg)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// GitHub App auth (optional): one credential for both the status API and
	// the git clone. Falls back to the PAT when not configured.
	var appClient *http.Client
	gitToken := func(context.Context) (string, error) { return cfg.Token, nil }
	if cfg.GitHubAppConfigured() {
		app, err := githubapp.New(cfg)
		if err != nil {
			return err
		}
		appClient = app.HTTPClient()
		gitToken = app.Token
		slog.Info("using github app authentication", "client_id", cfg.GitHubAppClientID)
	}

	writer, err := provider.New(cfg, appClient)
	if err != nil {
		return err
	}
	puller, err := talos.New(ctx, cfg)
	if err != nil {
		return err
	}
	defer func() { _ = puller.Close() }()

	srv := server.New(ctx, cfg, writer, render.New(cfg), puller, gitToken)

	httpSrv := &http.Server{
		Addr:              cfg.Addr,
		Handler:           srv.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
	}
	go func() {
		<-ctx.Done()
		shutCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		_ = httpSrv.Shutdown(shutCtx)
	}()

	slog.Info("downflate listening",
		"version", version,
		"commit", commit,
		"addr", cfg.Addr,
		"forge", cfg.Forge,
		"repo", cfg.Path,
		"cluster_path", cfg.ClusterPath,
		"nodes", puller.Nodes(),
		"namespace", cfg.Namespace,
		"webhooks", cfg.WebhookSecret != "",
	)
	if err := httpSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	srv.Shutdown() // drain in-flight jobs
	slog.Info("downflate stopped")
	return nil
}

func setupLogging(cfg *config.Config) {
	opts := &slog.HandlerOptions{Level: cfg.LogLevel}
	var h slog.Handler
	if cfg.LogFormat == "json" {
		h = slog.NewJSONHandler(os.Stderr, opts)
	} else {
		h = slog.NewTextHandler(os.Stderr, opts)
	}
	slog.SetDefault(slog.New(h))
}
