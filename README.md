# complete-devops-implementation

A Go web application rebuilt as a full DevSecOps reference pipeline — every stage from source commit to a running pod is covered by an automated security control, and deployment follows GitOps via ArgoCD onto a Kubernetes (kind) cluster.

This repo exists primarily as a **portfolio / learning project**: each stage was added incrementally, with every tool choice deliberate and every security finding fixed (or consciously accepted and documented) rather than suppressed.

## Architecture

```
Developer → GitHub (push/PR)
                │
                ▼
      GitHub Actions CI (14 gated stages)
                │
                ▼
         Docker Hub (signed image + SBOM)
                │
                ▼
   helm/go-web-app-chart/values.yaml (tag bumped by CI)
                │
                ▼
        ArgoCD (autosync, self-heal)
                │
                ▼
   Kubernetes (kind) on EC2 — NGINX Ingress
```

CI and CD are deliberately decoupled: GitHub Actions never talks to the cluster directly. CI's only job is to produce a verified, signed image and update a Helm value; ArgoCD does the rest by continuously reconciling the cluster against what's committed in git.

## Tech stack

| Layer | Tools |
|---|---|
| Application | Go 1.25, stdlib `net/http` |
| Containerization | Docker (multi-stage build → `distroless/base-debian12:nonroot`) |
| CI | GitHub Actions |
| Security scanning | Gitleaks (secrets), gosec (SAST), govulncheck + Trivy (SCA), Trivy (license, container image, IaC config), Hadolint (Dockerfile), Checkov (Helm/K8s) |
| Supply chain | Syft (SBOM), Cosign (keyless signing + attestation via Sigstore/OIDC) |
| Registry | Docker Hub |
| Deployment | Helm 3, ArgoCD (GitOps, autosync + self-heal) |
| Cluster | Kubernetes via `kind`, running on a single AWS EC2 instance (ap-south-1) |
| Ingress | NGINX Ingress Controller |

## Repository structure

```
.
├── .github/workflows/        # CI pipeline (ci.yml)
├── helm/go-web-app-chart/    # Helm chart — single source of deployable truth
│   ├── Chart.yaml
│   ├── values.yaml           # image.tag bumped automatically by CI
│   └── templates/
├── argocd/
│   └── application.yaml      # ArgoCD Application (autosync + self-heal)
├── ingress-controller/nginx/ # Ingress controller config
├── k8s/manifests/            # Raw manifests (where not managed via Helm)
├── static/images/
├── Dockerfile
├── go.mod
└── main.go
```

## CI pipeline — 14 stages

Every stage gates the next; nothing reaches Docker Hub or the cluster without clearing the full chain.

| # | Stage | Tool |
|---|---|---|
| 0 | Secrets scan | Gitleaks |
| 1 | Build & unit test | `go build`, `go test -race -coverprofile` |
| 2 | Code quality | golangci-lint |
| 3 | SAST | gosec → SARIF → Security tab |
| 4 | SCA | govulncheck + Trivy (filesystem) |
| 5 | License compliance | Trivy (license scan) |
| 6 | Dockerfile lint | Hadolint |
| 7 | IaC scan | Checkov (Helm/K8s) + Trivy (config) |
| 8 | Build & push image | Docker Buildx → Docker Hub |
| 9 | SBOM generation | Syft (CycloneDX) |
| 10 | Container image scan | Trivy (scans the **pushed** image, not a rebuild) |
| 11 | Image signing | Cosign (keyless, GitHub OIDC) |
| 12 | Attestation | Cosign attest (SBOM predicate) |
| 13 | Update Helm tag | `sed` + git commit → triggers ArgoCD sync |

## Required GitHub secrets

| Secret | Used by | Notes |
|---|---|---|
| `DOCKERHUB_USERNAME` | Stages 8–12 | Docker Hub account username |
| `DOCKERHUB_TOKEN` | Stages 8–12 | Docker Hub **access token**, not account password |
| `TOKEN` | Stage 13 | Fine-grained GitHub PAT, Contents: Read & Write — needed because the default `GITHUB_TOKEN` can be blocked from triggering further automation on some repo settings |

No secret is needed for Cosign signing — keyless signing uses GitHub Actions' own OIDC identity (`id-token: write` permission), not a stored key.

## Local development

```bash
go build -o main .
go test ./... -race -coverprofile=coverage.out -covermode=atomic
gofmt -l .
go vet ./...
golangci-lint run ./... --timeout=5m
```

Security tools used locally before every push (same tools CI runs):

```bash
gitleaks detect --source . --verbose
gosec ./...
govulncheck ./...
trivy fs --severity CRITICAL,HIGH --exit-code 1 --ignore-unfixed .
trivy config --severity CRITICAL,HIGH .
hadolint Dockerfile
```

## Deployment

The cluster is bootstrapped once (kind + Helm + ArgoCD + NGINX Ingress on a single EC2 instance); after that, every deploy is a git commit.

```bash
# One-time: register the app with ArgoCD
argocd proj create go-web-app-project \
  --src "https://github.com/<your-username>/complete-devops-implementation.git" \
  --dest "https://kubernetes.default.svc,go-web-app"

kubectl apply -f argocd/application.yaml
```

From then on, `syncPolicy.automated` in `argocd/application.yaml` means:
- any commit to `helm/go-web-app-chart/` auto-deploys (no manual `argocd app sync` needed)
- any manual `kubectl edit` on the live cluster gets auto-reverted to match git (`selfHeal: true`)

## Known, consciously-accepted gaps

Documented here rather than silently fixed or ignored — this is itself part of the portfolio story:

- **Image pinned by mutable tag, not digest** (Checkov `CKV_K8S_43`): the tag is `github.run_id`, which is unique per build and never reused in practice, so this is functionally equivalent to digest-pinning today. True digest-pinning (resolving and templating the SHA into `values.yaml`) is a planned follow-up once an image-updater tool is introduced.
- **`soft_fail: true` on the IaC scan stage**: kept permissive while the Helm chart findings were being triaged one by one. Flip to `exit-code: "1"` / `soft_fail: false` once the chart has been fully reviewed.



