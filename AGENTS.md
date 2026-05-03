# Repository Guidelines

## Project Structure & Module Organization
`cmd/rdpgw` contains the main gateway binary and its packages (`config`, `web`, `protocol`, `transport`, `security`, `identity`, `rdp`, `kdcproxy`). `cmd/auth` contains the local auth helper used for PAM and NTLM flows. Protobuf definitions live in `proto/`, with generated Go output under `shared/auth/`; treat `*.pb.go` files as generated artifacts. Supporting docs are in `docs/`, static assets in `assets/`, local sample config lives in `dev/docker/`, and Kubernetes starter manifests live in `k8s/`.

## Build, Test, and Development Commands
Use `make` for the default local build; it runs module tidy and builds both binaries into `bin/`. Use `make build` to rebuild without extra targets, `make test` to run `go test -cover -v ./...`, and `make mod` to refresh `go.mod` and `go.sum`. For container-based local testing, use the root `docker-compose.yml` with `docker compose up`; it consumes the published GHCR image and mounts the sample config from `dev/docker/rdpgw.yaml`. Kubernetes starter manifests live under `k8s/`.

## Coding Style & Naming Conventions
Follow standard Go formatting with `gofmt` before committing. Keep package names lowercase, exported identifiers in Go-style PascalCase, and file names descriptive and lowercase where possible. Match existing package boundaries instead of adding cross-package helpers casually; most tests already live beside the code they cover.

## Testing Guidelines
Place tests in `*_test.go` files next to the target package. Add or update coverage for protocol parsing, auth flows, and web handlers when behavior changes. Prefer table-driven tests where multiple request or config permutations are involved. On Ubuntu-like systems, PAM-related builds may require `libpam-dev`, matching CI.

## Commit & Pull Request Guidelines
Recent history mixes short imperative subjects (`Add webinterface`) with conventional prefixes (`fix: improve ios compatibility`). Prefer concise imperative commit messages under about 72 characters and keep each commit focused. Pull requests should summarize behavior changes, note config or auth impacts, link issues when relevant, and include screenshots only for template or web UI changes.

## Security & Configuration Tips
Do not commit real certificates, session keys, or NTLM credentials. Keep local config in untracked files where possible, and review any change touching TLS, cookies, auth headers, sample config in `dev/docker/`, or Kubernetes sample config in `k8s/`.
