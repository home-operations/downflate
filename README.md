# downflate

Pre-pull a pull request's **changed container images** onto a [Talos Linux](https://www.talos.dev)
cluster *before* the PR merges, and report the result back as a commit status.

A GitOps PR that bumps an image tag can stall or fail at merge time while every
node pulls the new image cold. downflate renders the PR diff, extracts only the
images the change introduces, pulls them into each node's image cache ahead of
time, and posts a `success`/`failure` status so the author knows the images are
pullable before they hit `main`.

It is a minimal cousin of [`konflate`](https://github.com/home-operations/konflate):
it reuses [`flate`](https://github.com/home-operations/flate) as the render
engine and konflate's webhook/status patterns, but drops the UI/comments/MCP and
**adds the step konflate doesn't have** — actually pulling the images via the
Talos machinery API.

## How it works

```
POST /hooks ─▶ verify HMAC ─▶ parse PR ─▶ coalesce per-PR ─▶ commit status: pending
                                                                    │
                  flate RenderTrees(base, head)  ◀── shallow-clone base + PR head
                                                                    │
                  changed images = head image set ∖ base image set  (image.Extract/Split)
                                                                    │
                  Talos ImageService.Pull(NS_CRI) on every node     │
                                                                    ▼
                                                     commit status: success | failure
```

The webhook handler returns `202` immediately and processes asynchronously;
bursts of events for the same PR collapse to a single in-flight run
(latest-wins), bounded by `DOWNFLATE_CONCURRENCY`.

## Endpoints

| Method | Path       | Purpose                                            |
|--------|------------|----------------------------------------------------|
| `POST` | `/hooks`   | Forge webhook intake (`501` if no secret is set)   |
| `GET`  | `/healthz` | Liveness probe                                     |

## Configuration

All configuration is via environment variables.

| Variable | Required | Default | Description |
|---|---|---|---|
| `DOWNFLATE_REPO` | ✓ | — | Forge URI: `github://owner/repo`, `gitlab://gitlab.example.com/group/sub/project`, `forgejo://git.example.com/owner/repo` |
| `DOWNFLATE_TOKEN` | ✓* | — | Forge API token (commit-status write + private clone). *Optional if a GitHub App is configured. |
| `DOWNFLATE_TALOSCONFIG` | — | `/var/run/secrets/talos.dev/config` | Path to a talosconfig (mTLS creds + endpoints) |
| `DOWNFLATE_WEBHOOK_SECRET` | — | — | HMAC/token secret; **`/hooks` returns `501` until set** |
| `DOWNFLATE_GITHUB_APP_CLIENT_ID` | — | — | GitHub App **client ID** — enables App auth (GitHub only) |
| `DOWNFLATE_GITHUB_APP_PRIVATE_KEY` | — | — | App private key: PEM inline, or `@/path/to/key.pem` |
| `DOWNFLATE_CLUSTER_PATH` | — | repo root | Sub-path flate scans (e.g. `kubernetes`) |
| `DOWNFLATE_NODES` | — | talosconfig nodes/endpoints | Comma-separated node addresses to pull onto |
| `DOWNFLATE_IMAGE_NAMESPACE` | — | `cri` | `cri` (k8s workloads) or `system` |
| `DOWNFLATE_TALOS_CONTEXT` | — | current | talosconfig context name |
| `DOWNFLATE_GIT_DEPTH` | — | `1` | Shallow clone depth (`0` = full history) |
| `DOWNFLATE_TIMEOUT` | — | `15m` | Per-PR render+pull deadline |
| `DOWNFLATE_CONCURRENCY` | — | `2` | Max PRs (and pulls) processed in parallel |
| `DOWNFLATE_RESTRICT_EGRESS` | — | `false` | Enable flate's SSRF guard (set for untrusted/fork PRs) |
| `DOWNFLATE_STATUS_CONTEXT` | — | `Downflate` | Commit-status context label |
| `DOWNFLATE_CACHE_DIR` | — | flate default | flate on-disk source cache |
| `DOWNFLATE_ADDR` | — | `:8080` | HTTP listen address |
| `DOWNFLATE_LOG_LEVEL` | — | `info` | `debug`/`info`/`warn`/`error` |
| `DOWNFLATE_LOG_FORMAT` | — | `text` | `text` or `json` |

### Authentication

A **personal/project access token** (`DOWNFLATE_TOKEN`) is the simplest option and
works on all three forges — it needs commit-status write (`repo:status` on
GitHub) and read access to clone the repo.

On GitHub you can instead use a **GitHub App** (the konflate model). Set
`DOWNFLATE_GITHUB_APP_CLIENT_ID` (the App's client ID) and
`DOWNFLATE_GITHUB_APP_PRIVATE_KEY`; downflate signs an App JWT with the client ID
as issuer, discovers the repository installation, and mints short-lived
installation tokens used for **both** the commit-status API and the git clone, so
no static PAT is needed. The App needs *Commit statuses: write* and
*Contents: read* permissions.
(GitHub Apps are only required for *check runs*, which downflate does not use —
plain commit statuses work with either auth method.)

### Webhook setup

Point your forge's pull-request / merge-request webhook at `https://<host>/hooks`:

- **GitHub** — content type `application/json`, secret = `DOWNFLATE_WEBHOOK_SECRET`, event *Pull requests* (sends `X-Hub-Signature-256`).
- **Forgejo/Gitea** — secret = `DOWNFLATE_WEBHOOK_SECRET`, event *Pull Request* (sends `X-Gitea-Signature`).
- **GitLab** — secret token = `DOWNFLATE_WEBHOOK_SECRET`, trigger *Merge request events* (sends `X-Gitlab-Token`).

## Build & run

```bash
go build -trimpath -ldflags "-s -w" -o downflate ./cmd/downflate

DOWNFLATE_REPO=github://owner/repo \
DOWNFLATE_TOKEN=ghp_xxx \
DOWNFLATE_WEBHOOK_SECRET=$(openssl rand -hex 20) \
DOWNFLATE_TALOSCONFIG=/etc/talos/config \
DOWNFLATE_CLUSTER_PATH=kubernetes \
./downflate
```

The binary is fully self-contained (git is handled in-process via go-git — no
system `git` needed at runtime), so it runs on a distroless/scratch image.

## Notes

- **Namespace** — `cri` pulls into the Kubernetes (`k8s.io`) containerd
  namespace so kubelet sees the warmed image; use `system` only for
  Talos-managed system images.
- **Requires Talos ≥ 1.13** for the `ImageService.Pull` API.
- Importing flate as a library brings its full render stack (Helm v4, Flux,
  kustomize, k8s.io) into the binary — expect a large (~90 MB stripped)
  artifact in exchange for a single self-contained process.
