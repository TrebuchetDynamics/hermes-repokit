# Security policy

RepoKit builds containers, keeps provider credentials under a repository's
`.hermes`, gives agents terminals and connects them to messaging channels, so
we treat security reports as a priority.

## Reporting a vulnerability

Report privately through GitHub: open the repository's **Security** tab and
choose **Report a vulnerability**
(<https://github.com/TrebuchetDynamics/hermes-repokit/security/advisories/new>).
Do not open a public issue, pull request or discussion for a suspected
vulnerability.

Please include the RepoKit version or commit, the host OS, what an attacker
controls, the impact, and steps to reproduce. Never send real credentials,
tokens or `.hermes` contents; redact them.

We aim to acknowledge a report within 7 days and to agree on a disclosure date
with you once a fix is ready.

## Supported versions

RepoKit is pre-1.0. Fixes land on `main` and in the next release; only the
latest release is supported.

## Scope

In scope: the `repokit` CLI, `install.sh`, the generated Compose, build recipe,
launcher and development image, and how RepoKit configures Hermes profiles and
Kanban.

Out of scope, report upstream instead: Hermes Agent itself
(<https://github.com/NousResearch/hermes-agent>), model providers, messaging
platforms, and the third-party toolchains RepoKit pins.

## Security posture worth knowing

By default RepoKit runs the team without approval prompts: workers change the
repository and run commands unattended, from any connected chat. `setup` states
this and lets you keep Hermes's prompts instead. Hermes's hard-deny floor
applies either way. See [the team model](docs/team-model.md).
