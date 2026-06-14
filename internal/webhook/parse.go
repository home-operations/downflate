package webhook

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/home-operations/downflate/internal/config"
)

// Event is the normalized pull/merge-request payload downflate acts on.
type Event struct {
	Number  int    // PR/MR number (GitHub/Forgejo) or iid (GitLab)
	HeadSHA string // head commit SHA
	Base    string // base/target branch name
	Action  string // raw forge action, for logging
}

// HeadRef returns the server-side ref that resolves to the PR head, used to
// fetch the head tree without needing push access to a branch.
func (e Event) HeadRef(forge config.Forge) string {
	if forge == config.ForgeGitLab {
		return fmt.Sprintf("refs/merge-requests/%d/head", e.Number)
	}
	return fmt.Sprintf("refs/pull/%d/head", e.Number)
}

// Header names identifying the event kind per forge.
const (
	headerGitHubEvent  = "X-GitHub-Event"
	headerForgejoEvent = "X-Gitea-Event"
	headerGitLabEvent  = "X-Gitlab-Event"
)

// Parse extracts a pull/merge-request Event from the webhook body. ok is false
// (with a nil error) when the event is well-formed but not a PR action worth
// rendering — e.g. a closed/merged PR, or an unrelated event type.
func Parse(forge config.Forge, h http.Header, body []byte) (ev Event, ok bool, err error) {
	switch forge {
	case config.ForgeGitLab:
		return parseGitLab(h.Get(headerGitLabEvent), body)
	case config.ForgeForgejo:
		return parsePullRequest(h.Get(headerForgejoEvent), body)
	default:
		return parsePullRequest(h.Get(headerGitHubEvent), body)
	}
}

// parsePullRequest handles GitHub/Forgejo "pull_request" payloads, which share
// a shape: a top-level action + number, and a nested pull_request object.
func parsePullRequest(event string, body []byte) (Event, bool, error) {
	if event != "pull_request" {
		return Event{}, false, nil
	}
	var p struct {
		Action      string `json:"action"`
		Number      int    `json:"number"`
		PullRequest struct {
			Number int `json:"number"`
			Head   struct {
				SHA string `json:"sha"`
			} `json:"head"`
			Base struct {
				Ref string `json:"ref"`
			} `json:"base"`
		} `json:"pull_request"`
	}
	if err := json.Unmarshal(body, &p); err != nil {
		return Event{}, false, fmt.Errorf("parse pull_request: %w", err)
	}
	if !shouldProcess(p.Action) {
		return Event{}, false, nil
	}
	num := p.Number
	if num == 0 {
		num = p.PullRequest.Number
	}
	ev := Event{
		Number:  num,
		HeadSHA: p.PullRequest.Head.SHA,
		Base:    p.PullRequest.Base.Ref,
		Action:  p.Action,
	}
	if ev.Number == 0 || ev.HeadSHA == "" {
		return Event{}, false, nil
	}
	return ev, true, nil
}

// parseGitLab handles "Merge Request Hook" payloads.
func parseGitLab(event string, body []byte) (Event, bool, error) {
	if event != "Merge Request Hook" {
		return Event{}, false, nil
	}
	var p struct {
		ObjectAttributes struct {
			IID          int    `json:"iid"`
			Action       string `json:"action"`
			TargetBranch string `json:"target_branch"`
			LastCommit   struct {
				ID string `json:"id"`
			} `json:"last_commit"`
		} `json:"object_attributes"`
	}
	if err := json.Unmarshal(body, &p); err != nil {
		return Event{}, false, fmt.Errorf("parse merge_request: %w", err)
	}
	a := p.ObjectAttributes
	if !shouldProcessGitLab(a.Action) {
		return Event{}, false, nil
	}
	ev := Event{
		Number:  a.IID,
		HeadSHA: a.LastCommit.ID,
		Base:    a.TargetBranch,
		Action:  a.Action,
	}
	if ev.Number == 0 || ev.HeadSHA == "" {
		return Event{}, false, nil
	}
	return ev, true, nil
}

// shouldProcess reports whether a GitHub/Forgejo PR action introduces commits
// worth re-rendering. An empty action (some Forgejo payloads omit it) is
// treated as processable.
func shouldProcess(action string) bool {
	switch action {
	case "", "opened", "reopened", "synchronize", "synchronized", "edited":
		return true
	default:
		return false
	}
}

func shouldProcessGitLab(action string) bool {
	switch action {
	case "", "open", "reopen", "update":
		return true
	default:
		return false
	}
}
