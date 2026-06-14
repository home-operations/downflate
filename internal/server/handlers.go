package server

import (
	"io"
	"net/http"

	"github.com/home-operations/downflate/internal/webhook"
)

// maxBody caps the webhook request body size.
const maxBody = 25 << 20 // 25 MiB

// Handler returns the HTTP routes downflate serves.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /hooks", s.handleWebhook)
	mux.HandleFunc("GET /healthz", s.handleHealth)
	return mux
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = io.WriteString(w, "ok\n")
}

func (s *Server) handleWebhook(w http.ResponseWriter, r *http.Request) {
	if s.cfg.WebhookSecret == "" {
		http.Error(w, "webhooks disabled", http.StatusNotImplemented)
		return
	}
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxBody))
	if err != nil {
		http.Error(w, "cannot read body", http.StatusBadRequest)
		return
	}
	if err := webhook.Verify(s.cfg.Forge, s.cfg.WebhookSecret, r.Header, body); err != nil {
		s.log.Warn("webhook verification failed", "err", err, "remote", r.RemoteAddr)
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	ev, ok, err := webhook.Parse(s.cfg.Forge, r.Header, body)
	if err != nil {
		s.log.Warn("webhook parse failed", "err", err)
		http.Error(w, "bad payload", http.StatusBadRequest)
		return
	}
	if !ok {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	s.submit(ev)
	s.log.Info("accepted", "pr", ev.Number, "action", ev.Action, "sha", short(ev.HeadSHA))
	w.WriteHeader(http.StatusAccepted)
}
