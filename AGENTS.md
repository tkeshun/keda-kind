# Repository Guidelines

## Project Structure & Module Organization
`sample-app/` contains the Go applications: `cmd/enqueue` and `cmd/dequeue` are the entrypoints, and `internal/` holds shared config, queue logic, message handling, and tests. `manifest/` contains Helm charts for `enqueue-app`, `dequeue-app`, `elasticmq`, `postgresql`, and the KEDA wrapper chart. Keep design specs in `docs/superpowers/specs/` and implementation plans in `docs/superpowers/plans/`. Top-level operational files include `Makefile`, `compose.yaml`, and `kind-config.yaml`.

## Build, Test, and Development Commands
Use the `Makefile` as the main entrypoint.

- `make test`: run `go test ./...` with repo-local caches.
- `make build`: build `local/enqueue:dev` and `local/dequeue:dev` with Docker Buildx.
- `make kind-create`: create the local kind cluster using `.cache/kubeconfig`.
- `make helm-deps`: refresh Helm dependencies for the KEDA wrapper chart.
- `make install-elasticmq install-postgresql install-keda`: install infrastructure charts.
- `make install-enqueue install-dequeue`: deploy the sample apps with develop values.
- `make compose-up` / `make compose-run-dequeue`: run the non-KEDA local validation flow.

## Coding Style & Naming Conventions
Follow standard Go formatting with `gofmt`; keep packages small and focused under `sample-app/internal/`. Use tabs as emitted by Go tools. Prefer descriptive package names such as `config`, `enqueue`, and `dequeue`. For Helm, keep chart names, release names, and value keys aligned with the existing `manifest/*` patterns. Put environment-specific overrides in `values/develop.yaml` or `values/production.yaml` instead of hardcoding.

## Testing Guidelines
Add Go tests next to the code as `*_test.go`. Prefer targeted package tests while iterating, then run `make test` before finishing. Keep regression coverage for repo wiring too; `sample-app/layout/layout_test.go` is the model for tests that protect file paths and build references.

## Commit & Pull Request Guidelines
Match the commit style already used in history: `docs: ...`, `fix: ...`, `test: ...`, `refactor: ...`, `feat: ...`. Keep commits scoped to one change. PRs should explain the behavior change, list verification commands, and call out any chart or environment-variable impact. Include logs or screenshots only when they clarify runtime behavior.

## Agent-Specific Instructions
If you are using Codex or another coding agent in this repository, use superpower skills first. When a spec is needed, write it in Japanese under `docs/superpowers/specs/` before implementation. When presenting options, include both merits and drawbacks, and explain the intent of any command before running it.
ただし、軽微な変更な場合は適宜省略してもよい。