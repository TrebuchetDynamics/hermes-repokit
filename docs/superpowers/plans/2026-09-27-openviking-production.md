> Historical sidecar implementation record. Current normal deployment embeds memory in Hermes; see [embedded topology](../../qualification/embedded-openviking.md). Earlier fixture results do not qualify the current image.

# OpenViking production integration

Execute the existing bootstrap design's Phase 9 before live team
acceptance, per the latest user directive. Work in feat/openviking-production;
preserve the active root deployment and the separate runtime-integrations worktree.

## Constraints

Go bootstrapper; native upstream memory provider; official pinned OpenViking;
no public port, model guesses, credential copies or runtime RepoKit callback.
Keep all six roles and manual Kanban dispatch unchanged. Private native setup
belongs in the operator's terminal. Metadata verification performs no inference.

## Implementation steps

1. Render the pinned official OpenViking service with the host UID/GID, explicit
   private HOME/config path, persistent /app/.openviking, native healthcheck and
   no bot. Generate its private directory before Compose startup. Test exact
   service contract and preservation of Hermes image entrypoint/user semantics.
2. Make normal fresh install include OpenViking. Upgrade only an exact recognized
   Hermes-only Compose under the existing writer lock; preserve native config,
   launcher, profiles and owner files. Keep a private old-Compose backup. Refuse
   edited/ambiguous files and an unexplained pre-existing sidecar directory.
   Test fresh/rerun/upgrade, refusal, interruption and owner-data preservation.
3. Add native OpenViking init/doctor setup delegation with inherited terminal
   streams, explicit Compose/context and no automatic startup. Offer native
   Hermes memory setup for the shared connection. Do not capture credentials.
4. Add bounded read-only OpenViking service/config metadata observations, keeping
   live memory acceptance explicitly unknown. Test absent/unconfigured/stopped/
   wrong-image/wrong-mount and healthy-runtime-but-unproven-memory cases.
5. Qualify the actual nonroot pinned sidecar pending mode and persistent mount in
   a disposable repository. Prove the old installation still reruns safely and
   the new artifacts survive removal of installer code. Run Go unit/race/vet.
6. After private embedding/extraction/server-auth configuration is available,
   qualify native shared connection for all six profiles, real cross-role recall,
   restart recall, second-repository denial and outage behavior. Then proceed to
   Real model workers and full removal-first acceptance. No synthetic
   response or mere health check counts as live memory acceptance.

## Review focus

- Existing Hermes-only deployments must not lose native state during upgrade.
- Unknown OpenViking state and edited Compose must be preserved, not adopted.
- New sidecar files must remain host-user-owned and private after native startup.
- No service start, provider activation or model call hidden in install/verify.
- A configured or healthy server is not evidence of memory recall or isolation.

## Execution ledger

Baseline: go test ./... passed at dd6c16e in this isolated worktree.
Ruling: reuse the already reviewed Phase 9 architecture and latest execution
order; no roster redesign or new approval cycle. Cost if wrong: revisit only
the integration handoff, not the already accepted six-role organization.
Ruling: use the already schema-qualified v0.4.21 digest; new evidence will
qualify pending-mode/runtime behavior, not assert live memory success. Cost if
wrong: select and requalify another immutable image before live activation.
Source finding: the image entrypoint supports pending configuration HTTP 503.
Source finding: native init ignores --help and launches its wizard. A disposable,
network-disabled, unmounted source probe exited without configuration; never
use --help to probe this wizard on a configured deployment.

Tasks 1–4: implemented. Fresh and existing deployment tests, private sidecar
rendering, native setup terminal/error contracts, metadata-only probes and
shared-link filesystem tests passed. Each initial behavior test was observed
failing before implementation. The native connection linker preserves unrelated
profile settings and preflights every candidate before specialist writes.

Task 5: complete for credential-free wiring. Pending-mode Docker fixture passed, including UID, 503, native path
helpers, rerun, source/binary removal and Compose recreation persistence.
The original native six-profile/Kanban foundation fixture passed with generated
OpenViking present but not started. The extended native connection test passed in 144.37 seconds, separately from
live memory; it uses a clearly nonsecret dummy connection and never performs an
API request. It exercises actual native setters and profile-scoped resolution
for all six cloned roles. Full offline/race suites, vet, formatting, diff and
local documentation-link checks passed.

Final review: two Important/P2 findings fixed in one pass. Unknown empty or
marker-only sidecar directories now refuse without a prior exact backup
(TestMemoryUpgradePreservesUnknownFiles RED→GREEN); known interrupted
preparation remains resumable. Matching running containers are now separated
from runtime health so native doctor can diagnose starting/unhealthy services
(TestOpenVikingObservationNeverProbesModels RED→GREEN). A stopped/missing Hermes
preflight also has a RED→GREEN regression.

Source finding: native init honors OPENVIKING_CONFIG_FILE, writes mode 0600
atomically, and defaults its workspace to the neighboring data directory.
Source finding: /ready performs embedding inference; metadata verify and the
native /health healthcheck avoid it.
Source finding: the Hermes native serializer accepts account/user, but the
resolver deliberately omits their assertion headers for user keys. The server
binds identity to the key. The native fixture's identity assertion was corrected
to this behavior; production already checks server-returned identity.

Task 6: waiting for the operator's embedding/extraction model endpoints and
private native configuration. Requested via asynchronous input. No credentials
borrowed, no guessed model, no live-memory success inferred from fixture data.
Actual workers and full removal-first dogfood remain after this gate.
Implementation was validated in an isolated worktree. Delivery does not upgrade
the active main deployment or include the unrelated runtime-integrations work.
