package webhook

import (
	"net/http"
	"testing"

	"github.com/home-operations/downflate/internal/config"
)

// header builds a canonicalized header set, matching how Go normalizes
// incoming request headers (so h.Get works regardless of source casing).
func header(kv ...string) http.Header {
	h := http.Header{}
	for i := 0; i+1 < len(kv); i += 2 {
		h.Set(kv[i], kv[i+1])
	}
	return h
}

func TestParseGitHubPullRequest(t *testing.T) {
	body := []byte(`{
		"action": "synchronize",
		"number": 42,
		"pull_request": {
			"head": {"sha": "deadbeef", "ref": "feature"},
			"base": {"ref": "main"}
		}
	}`)
	h := header(headerGitHubEvent, "pull_request")
	ev, ok, err := Parse(config.ForgeGitHub, h, body)
	if err != nil || !ok {
		t.Fatalf("want ok, got ok=%v err=%v", ok, err)
	}
	if ev.Number != 42 || ev.HeadSHA != "deadbeef" || ev.Base != "main" {
		t.Fatalf("unexpected event: %+v", ev)
	}
	if got := ev.HeadRef(config.ForgeGitHub); got != "refs/pull/42/head" {
		t.Fatalf("HeadRef = %q", got)
	}
}

func TestParseGitHubClosedIgnored(t *testing.T) {
	body := []byte(`{"action":"closed","number":7,"pull_request":{"head":{"sha":"x"},"base":{"ref":"main"}}}`)
	h := header(headerGitHubEvent, "pull_request")
	_, ok, err := Parse(config.ForgeGitHub, h, body)
	if err != nil || ok {
		t.Fatalf("closed PR should be ignored: ok=%v err=%v", ok, err)
	}
}

func TestParseNonPullRequestEventIgnored(t *testing.T) {
	h := header(headerGitHubEvent, "push")
	_, ok, err := Parse(config.ForgeGitHub, h, []byte(`{}`))
	if err != nil || ok {
		t.Fatalf("push event should be ignored: ok=%v err=%v", ok, err)
	}
}

func TestParseGitLabMergeRequest(t *testing.T) {
	body := []byte(`{
		"object_kind": "merge_request",
		"object_attributes": {
			"iid": 13,
			"action": "update",
			"target_branch": "main",
			"last_commit": {"id": "cafe1234"}
		}
	}`)
	h := header(headerGitLabEvent, "Merge Request Hook")
	ev, ok, err := Parse(config.ForgeGitLab, h, body)
	if err != nil || !ok {
		t.Fatalf("want ok, got ok=%v err=%v", ok, err)
	}
	if ev.Number != 13 || ev.HeadSHA != "cafe1234" || ev.Base != "main" {
		t.Fatalf("unexpected event: %+v", ev)
	}
	if got := ev.HeadRef(config.ForgeGitLab); got != "refs/merge-requests/13/head" {
		t.Fatalf("HeadRef = %q", got)
	}
}
