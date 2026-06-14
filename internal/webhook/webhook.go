// Package webhook verifies and parses incoming forge pull-request webhooks.
package webhook

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"net/http"
	"strings"

	"github.com/home-operations/downflate/internal/config"
)

// Sentinel errors returned by Verify. Callers map any non-nil error to HTTP 401.
var (
	ErrNoSecret          = errors.New("webhook: no secret configured")
	ErrMissingSignature  = errors.New("webhook: missing signature header")
	ErrSignatureMismatch = errors.New("webhook: signature mismatch")
)

// Header names carrying the signature/token per forge.
const (
	headerGitHubSig  = "X-Hub-Signature-256" // value: "sha256=<hex>"
	headerForgejoSig = "X-Gitea-Signature"   // value: bare hex
	headerGitLabTok  = "X-Gitlab-Token"      // value: the shared secret verbatim
)

// Verify authenticates a webhook request body against the configured secret.
//
// GitHub and Forgejo use HMAC-SHA256 over the raw body; GitLab uses a plain
// shared-token comparison. All comparisons are constant-time.
func Verify(forge config.Forge, secret string, h http.Header, body []byte) error {
	if secret == "" {
		return ErrNoSecret
	}
	switch forge {
	case config.ForgeGitHub:
		return verifyHMAC(secret, body, h.Get(headerGitHubSig), "sha256=")
	case config.ForgeForgejo:
		return verifyHMAC(secret, body, h.Get(headerForgejoSig), "")
	case config.ForgeGitLab:
		return verifyToken(secret, h.Get(headerGitLabTok))
	default:
		return ErrSignatureMismatch
	}
}

func verifyHMAC(secret string, body []byte, sig, prefix string) error {
	if sig == "" {
		return ErrMissingSignature
	}
	sig = strings.TrimPrefix(sig, prefix)
	got, err := hex.DecodeString(sig)
	if err != nil {
		return ErrSignatureMismatch
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	if !hmac.Equal(got, mac.Sum(nil)) {
		return ErrSignatureMismatch
	}
	return nil
}

func verifyToken(secret, tok string) error {
	if tok == "" {
		return ErrMissingSignature
	}
	if subtle.ConstantTimeCompare([]byte(secret), []byte(tok)) != 1 {
		return ErrSignatureMismatch
	}
	return nil
}
