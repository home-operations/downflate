package server

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/home-operations/downflate/internal/config"
	"github.com/home-operations/downflate/internal/git"
	"github.com/home-operations/downflate/internal/provider"
	"github.com/home-operations/downflate/internal/render"
	"github.com/home-operations/downflate/internal/talos"
)

type fakeWriter struct {
	mu     sync.Mutex
	states []provider.State
}

func (f *fakeWriter) SetStatus(_ context.Context, st provider.Status) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.states = append(f.states, st.State)
	return nil
}

func (f *fakeWriter) seen() []provider.State {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]provider.State(nil), f.states...)
}

type fakeRenderer struct {
	calls atomic.Int64
}

func (r *fakeRenderer) Render(_ context.Context, _, _ string) (*render.Result, error) {
	r.calls.Add(1)
	return &render.Result{}, nil // no manifests ⇒ no changed images
}

type fakePuller struct{}

func (fakePuller) PullAll(_ context.Context, _ []string) []talos.Result { return nil }
func (fakePuller) Nodes() []string                                      { return []string{"10.0.0.1"} }

func testServer(t *testing.T, secret string, w provider.Writer, r Renderer) *Server {
	t.Helper()
	cfg := &config.Config{
		Forge:         config.ForgeGitHub,
		WebhookSecret: secret,
		Concurrency:   1,
		Timeout:       5 * time.Second,
		StatusContext: "downflate",
	}
	srv := New(context.Background(), cfg, w, r, fakePuller{}, nil)
	// Inject a no-op materializer so tests never touch the network/git.
	srv.materialize = func(_ context.Context, _, _ string) (*git.Trees, error) {
		return &git.Trees{}, nil
	}
	return srv
}

func ghSig(secret string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

func post(srv *Server, body string, hdr map[string]string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/hooks", strings.NewReader(body))
	for k, v := range hdr {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	return rec
}

func TestWebhookDisabled(t *testing.T) {
	srv := testServer(t, "", &fakeWriter{}, &fakeRenderer{})
	rec := post(srv, "{}", nil)
	if rec.Code != http.StatusNotImplemented {
		t.Fatalf("want 501, got %d", rec.Code)
	}
}

func TestWebhookBadSignature(t *testing.T) {
	srv := testServer(t, "secret", &fakeWriter{}, &fakeRenderer{})
	rec := post(srv, "{}", map[string]string{
		"X-GitHub-Event":      "pull_request",
		"X-Hub-Signature-256": "sha256=deadbeef",
	})
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("want 401, got %d", rec.Code)
	}
}

func TestWebhookNonPRIgnored(t *testing.T) {
	secret := "secret"
	srv := testServer(t, secret, &fakeWriter{}, &fakeRenderer{})
	body := `{}`
	rec := post(srv, body, map[string]string{
		"X-GitHub-Event":      "push",
		"X-Hub-Signature-256": ghSig(secret, []byte(body)),
	})
	if rec.Code != http.StatusNoContent {
		t.Fatalf("want 204, got %d", rec.Code)
	}
}

func TestWebhookAcceptedAndProcessed(t *testing.T) {
	secret := "secret"
	w := &fakeWriter{}
	r := &fakeRenderer{}
	srv := testServer(t, secret, w, r)

	body := `{"action":"opened","number":5,"pull_request":{"head":{"sha":"abc123"},"base":{"ref":"main"}}}`
	rec := post(srv, body, map[string]string{
		"X-GitHub-Event":      "pull_request",
		"X-Hub-Signature-256": ghSig(secret, []byte(body)),
	})
	if rec.Code != http.StatusAccepted {
		t.Fatalf("want 202, got %d", rec.Code)
	}

	srv.Shutdown() // wait for the async job

	if r.calls.Load() != 1 {
		t.Errorf("renderer called %d times, want 1", r.calls.Load())
	}
	states := w.seen()
	// pending first, then success (no changed images path).
	if len(states) < 2 || states[0] != provider.Pending || states[len(states)-1] != provider.Success {
		t.Errorf("status sequence = %v, want [pending ... success]", states)
	}
}
