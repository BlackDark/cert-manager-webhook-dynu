# cert-manager-webhook-dynu

Cert-manager DNS01 webhook backed by the dynu API, with a Helm deployment chart.

## Verify

`mise run check`: gofmt, vet, build, all unit tests, and Helm lint.

Integration tests use the `integration` build tag and real DNS credentials; do not run them without explicit test-zone details. CI uses the required `check` gate plus actionlint, zizmor, and strict Renovate validation.

## Stack

Follows https://github.com/BlackDark/agent-presets/blob/main/stack/core.md. Existing image and chart release workflows remain unchanged.

## Map

- `main.go`, `main_test.go`: webhook setup and solver registration.
- `dynu/`: dynu API client and tests.
- `deploy/dynu-webhook/`: Helm chart.
- `integration_test.go`: live ACME DNS01 suite behind the `integration` tag.

## Git

- Conventional Commits, one topic per PR.
- Squash unless commits need separate review.
- Force-push only with `--force-with-lease` on own branches.

## Boundaries

- Tools come from `mise.toml`; no global installs.
- Do not create tags, publish images, or run live DNS integration tests from a local session.
