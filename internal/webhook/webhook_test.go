package webhook

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"testing"

	"github.com/home-operations/downflate/internal/config"
)

func sign(secret string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}

func TestVerifyGitHub(t *testing.T) {
	secret := "s3cret"
	body := []byte(`{"action":"opened"}`)

	t.Run("valid", func(t *testing.T) {
		h := http.Header{headerGitHubSig: {"sha256=" + sign(secret, body)}}
		if err := Verify(config.ForgeGitHub, secret, h, body); err != nil {
			t.Fatalf("want nil, got %v", err)
		}
	})
	t.Run("tampered body", func(t *testing.T) {
		h := http.Header{headerGitHubSig: {"sha256=" + sign(secret, body)}}
		if err := Verify(config.ForgeGitHub, secret, h, []byte("other")); err != ErrSignatureMismatch {
			t.Fatalf("want ErrSignatureMismatch, got %v", err)
		}
	})
	t.Run("wrong secret", func(t *testing.T) {
		h := http.Header{headerGitHubSig: {"sha256=" + sign("nope", body)}}
		if err := Verify(config.ForgeGitHub, secret, h, body); err != ErrSignatureMismatch {
			t.Fatalf("want ErrSignatureMismatch, got %v", err)
		}
	})
	t.Run("missing header", func(t *testing.T) {
		if err := Verify(config.ForgeGitHub, secret, http.Header{}, body); err != ErrMissingSignature {
			t.Fatalf("want ErrMissingSignature, got %v", err)
		}
	})
	t.Run("no secret", func(t *testing.T) {
		if err := Verify(config.ForgeGitHub, "", http.Header{}, body); err != ErrNoSecret {
			t.Fatalf("want ErrNoSecret, got %v", err)
		}
	})
}

func TestVerifyForgejo(t *testing.T) {
	secret := "abc"
	body := []byte(`{"action":"synchronized"}`)
	h := http.Header{headerForgejoSig: {sign(secret, body)}} // bare hex, no prefix
	if err := Verify(config.ForgeForgejo, secret, h, body); err != nil {
		t.Fatalf("valid forgejo sig: want nil, got %v", err)
	}
	bad := http.Header{headerForgejoSig: {sign("wrong", body)}}
	if err := Verify(config.ForgeForgejo, secret, bad, body); err != ErrSignatureMismatch {
		t.Fatalf("bad forgejo sig: want mismatch, got %v", err)
	}
}

func TestVerifyGitLab(t *testing.T) {
	secret := "tok"
	body := []byte(`{}`)
	if err := Verify(config.ForgeGitLab, secret, http.Header{headerGitLabTok: {"tok"}}, body); err != nil {
		t.Fatalf("valid token: want nil, got %v", err)
	}
	if err := Verify(config.ForgeGitLab, secret, http.Header{headerGitLabTok: {"bad"}}, body); err != ErrSignatureMismatch {
		t.Fatalf("bad token: want mismatch, got %v", err)
	}
	if err := Verify(config.ForgeGitLab, secret, http.Header{}, body); err != ErrMissingSignature {
		t.Fatalf("missing token: want ErrMissingSignature, got %v", err)
	}
}
