# Generic repository team implementation

The user's Universal Team Roster directive (six roles including steward) is the governing specification.
It supersedes builder/engineering identities in earlier runtime plans. Work in the
current checkout to preserve and extend the uncommitted native-bootstrap changes.
The original implementation scope excluded Git delivery. The subsequent explicit
commit/push/merge request authorizes Git delivery; deployment to this repository
and copying host credentials remain outside that scope.

1. Define the exact six-role roster, descriptions and common-plus-role SOULs in
   internal/team. Test identity, authority, artifact neutrality and same-card review.
2. Replace premature blank-profile provisioning with post-successful-native-setup
   configuration cloning. Use native profile creation, strip cloned curated memory
   only in new profiles, immediately replace newly cloned identity without renaming native directories, and preserve
   user-edited or unknown profiles. Keep generated identity fingerprints solely as
   installer reconciliation metadata, never runtime task truth.
3. Make conservative Kanban defaults explicit and give default the orchestration
   toolset. Qualify toolsets against the pinned image; report advisory boundaries.
4. Update plan/verify and documentation to make the generic team the default and
   distinguish provisioned files from the full acceptance gate.
5. Exercise native cloning, profile resolution, one board, parent metadata,
   same-card changes/approval, restart/recreation and installer removal in a
   disposable Docker repository. Test interrupted provisioning and drift offline.
6. Qualify OpenViking and Nerve/Laya before enabling them. Preserve shared
   account=repokit/user=repository-ID/no-peer semantics; no hosted Nerve bootstrap.
   Missing embedding/VLM/provider configuration or model qualification is pending,
   never a passing full-team result. Record all 15 acceptance outcomes explicitly.

Review focus: clone failure must not claim a complete team or erase already-created native profiles; retries must never
clear role memories; setup cancellation must never provision credential-bearing
profiles; default tools must not retain hermes-cli implementation tools; edited
profiles must stay byte-for-byte unchanged; no offline test may certify inference.

Validation: focused Go and filesystem tests, go test ./..., go test -race ./...,
go vet ./..., and REPOKIT_DOCKER_TESTS=1 go test -tags=docker ./tests/acceptance.

Steward owns native profile lifecycle; default remains primary conversational UX.
Skills first, retirement before deletion, explicit user authorization for deletion.

## Review resolutions

- Native profile creation registers services/routing. Use the final native name,
  immediately clear cloned identity, and preserve incomplete profiles on failure;
  never raw-rename or auto-delete a native profile to simulate a transaction.
- A coordinator may exist while specialists remain pending. No complete-team claim
  follows a failed clone. This preserves native state instead of rolling it back.
- Adopt default only if every generated config field and routing description is
  pristine, not merely SOUL. Otherwise report drift without overwriting it.
- Setup explicitly selects default; nonterminal success cannot authorize cloning.
- Native lifecycle tests record distinct profile processes/runs but do not prove
  malicious-caller resistance, model-driven dispatch or artifact verification.
