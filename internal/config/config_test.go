package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseRepo(t *testing.T) {
	tests := []struct {
		raw       string
		wantForge Forge
		wantHost  string
		wantOwner string
		wantRepo  string
		wantPath  string
		wantErr   bool
	}{
		{raw: "github://owner/repo", wantForge: ForgeGitHub, wantHost: "", wantOwner: "owner", wantRepo: "repo", wantPath: "owner/repo"},
		{raw: "github://ghe.example.com/org/proj", wantForge: ForgeGitHub, wantHost: "ghe.example.com", wantOwner: "org", wantRepo: "proj", wantPath: "org/proj"},
		{raw: "gitlab://gitlab.example.com/group/sub/project", wantForge: ForgeGitLab, wantHost: "gitlab.example.com", wantOwner: "group/sub", wantRepo: "project", wantPath: "group/sub/project"},
		{raw: "forgejo://git.example.com/owner/repo", wantForge: ForgeForgejo, wantHost: "git.example.com", wantOwner: "owner", wantRepo: "repo", wantPath: "owner/repo"},
		{raw: "bitbucket://owner/repo", wantErr: true},
		{raw: "github://owner", wantErr: true},
		{raw: "", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.raw, func(t *testing.T) {
			c := &Config{}
			err := c.parseRepo(tt.raw)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error for %q", tt.raw)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if c.Forge != tt.wantForge || c.Host != tt.wantHost || c.Owner != tt.wantOwner || c.Repo != tt.wantRepo || c.Path != tt.wantPath {
				t.Errorf("got forge=%q host=%q owner=%q repo=%q path=%q",
					c.Forge, c.Host, c.Owner, c.Repo, c.Path)
			}
		})
	}
}

func TestGitHubAppConfigured(t *testing.T) {
	if (&Config{}).GitHubAppConfigured() {
		t.Error("empty config should not be App-configured")
	}
	if (&Config{GitHubAppID: 1}).GitHubAppConfigured() {
		t.Error("app id without key should not be App-configured")
	}
	if !(&Config{GitHubAppID: 1, GitHubAppPrivateKey: []byte("k")}).GitHubAppConfigured() {
		t.Error("app id + key should be App-configured")
	}
}

func TestLoadAppPrivateKey(t *testing.T) {
	t.Run("inline", func(t *testing.T) {
		c := &Config{}
		if err := c.loadAppPrivateKey("PEM-INLINE"); err != nil {
			t.Fatal(err)
		}
		if string(c.GitHubAppPrivateKey) != "PEM-INLINE" {
			t.Fatalf("got %q", c.GitHubAppPrivateKey)
		}
	})
	t.Run("file", func(t *testing.T) {
		p := filepath.Join(t.TempDir(), "key.pem")
		if err := os.WriteFile(p, []byte("PEM-FILE"), 0o600); err != nil {
			t.Fatal(err)
		}
		c := &Config{}
		if err := c.loadAppPrivateKey("@" + p); err != nil {
			t.Fatal(err)
		}
		if string(c.GitHubAppPrivateKey) != "PEM-FILE" {
			t.Fatalf("got %q", c.GitHubAppPrivateKey)
		}
	})
	t.Run("missing file", func(t *testing.T) {
		c := &Config{}
		if err := c.loadAppPrivateKey("@/no/such/key.pem"); err == nil {
			t.Fatal("expected error for missing file")
		}
	})
	t.Run("empty", func(t *testing.T) {
		c := &Config{}
		if err := c.loadAppPrivateKey(""); err != nil || c.GitHubAppPrivateKey != nil {
			t.Fatalf("empty should be a no-op, got key=%q err=%v", c.GitHubAppPrivateKey, err)
		}
	})
}

func TestCloneURL(t *testing.T) {
	tests := []struct {
		raw  string
		want string
	}{
		{"github://owner/repo", "https://github.com/owner/repo.git"},
		{"github://ghe.example.com/org/proj", "https://ghe.example.com/org/proj.git"},
		{"gitlab://group/sub/project", "https://gitlab.com/group/sub/project.git"},
		{"forgejo://git.example.com/owner/repo", "https://git.example.com/owner/repo.git"},
	}
	for _, tt := range tests {
		c := &Config{}
		if err := c.parseRepo(tt.raw); err != nil {
			t.Fatalf("parseRepo(%q): %v", tt.raw, err)
		}
		if got := c.CloneURL(); got != tt.want {
			t.Errorf("CloneURL(%q) = %q, want %q", tt.raw, got, tt.want)
		}
	}
}
