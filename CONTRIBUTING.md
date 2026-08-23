# Contributing to CloudWeave

## Development Setup

```bash
git clone https://github.com/jhanvi857/CloudWeave.git
cd CloudWeave
go build -o node.exe ./cmd/node
go test ./...
```

## Docker Image Publishing

CloudWeave images are automatically built and published to [GitHub Container Registry (GHCR)](https://github.com/jhanvi857/CloudWeave/pkgs/container/cloudweave) via GitHub Actions on every push to `main` and on version tags.

### CI/CD Pipeline

The workflow (`.github/workflows/docker.yml`) runs a staged pipeline:

1. **Build** — Multi-architecture image (`linux/amd64` + `linux/arm64`) with build provenance and SBOM attestation.
2. **Trivy Scan** — Vulnerability scan; the job hard-fails on Critical severity findings.
3. **Smoke Test** — Pulls the image on both architectures (via QEMU emulation), starts a container with `--memory=256m`, and verifies `/health` returns 200 and no OOM-kill occurs.
4. **Promote** — Only on version tags (`v*.*.*`): retags the tested image as `latest` and semver (`1.2.3`, `1.2`).

### Tag Semantics

| Push type | Tags created |
|---|---|
| Push to `main` | `sha-<short>` only |
| Version tag `v1.2.3` | `sha-<short>`, `1.2.3`, `1.2`, `latest` |

`latest` only moves on version-tag releases — main-branch pushes get SHA tags so downstream users on `:latest` are never surprised by an untested intermediate commit.

### First-Run Setup (Repository Owner)

> [!IMPORTANT]
> After the first successful workflow run, the GHCR package is created as **Private** by default.
> You must manually change visibility to **Public** for anonymous `docker pull` to work:
>
> 1. Go to your GitHub repository → **Packages** (right sidebar).
> 2. Click on the `cloudweave` package.
> 3. Click **Package settings** (gear icon).
> 4. Under **Danger Zone**, click **Change visibility** → select **Public** → confirm.
>
> This is a one-time step. All subsequent pushes will publish to the now-public package automatically.
> **If you fork this repository**, you need to repeat this step in your fork.

### Creating a Release

```bash
git tag v1.0.0
git push origin v1.0.0
```

This triggers the full pipeline: build → scan → smoke test → promote to `latest` + `1.0.0` + `1.0`.

### Verifying a Published Image

From any machine (no GHCR auth required once visibility is public):

```bash
docker pull ghcr.io/jhanvi857/cloudweave:latest
docker run --rm -p 9000:9000 ghcr.io/jhanvi857/cloudweave:latest
curl http://localhost:9000/health   # Should return "OK"
```

## Running Tests

```bash
# All tests
go test ./...

# Integration tests only
go test ./test/integration/

# Benchmarks
go test -bench . -count=5 ./test/benchmark
```

## Code Style

- Go standard formatting (`gofmt`)
- All existing tests must pass before submitting changes
- New features should include tests and a brief doc section
