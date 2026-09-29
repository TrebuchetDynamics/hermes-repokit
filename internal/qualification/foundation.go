package qualification

// FoundationImage is the immutable candidate exercised by the credential-free
// Docker fixture. See docs/qualification/runtime-observations.md. This pin does
// not qualify authenticated chat, optional integrations or the full v1 release.
const FoundationImage = "nousresearch/hermes-agent@sha256:d4da4a40cd7a28aba983775d9fd31d94cbf153eeb0cb9e844d6d0f612b7c24db"

// NativeDefaultSoulSHA256 is the stock SOUL.md the pinned FoundationImage CLI
// writes into a fresh home (observed output, not a private import). A default
// profile still holding exactly this SOUL is unclaimed and may be adopted;
// requalify it whenever FoundationImage changes.
const NativeDefaultSoulSHA256 = "36c1f5a2e92cd1d018311eaf4c8f1e8886672eae78212e033c681d0e3d5d506f"

// HermesImportNames are the top-level Python modules the pinned FoundationImage
// runtime imports in gateway-spawned workers. Hermes launches workers as
// `python -m hermes_cli.main` from /workspace, and its editable install sits
// after the working directory on sys.path, so a repository-root module with one
// of these names shadows Hermes' own (observed: a root `tools/` package silently
// removed every file tool). Derived from the pinned image by loading the worker
// entry points and their static imports; requalify whenever FoundationImage changes.
var HermesImportNames = []string{
	"acp_adapter", "agent", "cli", "cron", "gateway", "hermes_bootstrap", "hermes_cli",
	"hermes_constants", "hermes_constants_scratch", "hermes_logging", "hermes_platform",
	"hermes_startup_watchdog", "hermes_state", "hermes_state_common", "hermes_state_compression",
	"hermes_state_dbfile", "hermes_state_errors", "hermes_state_fts", "hermes_state_gateway",
	"hermes_state_guard", "hermes_state_health", "hermes_state_holders", "hermes_state_ids",
	"hermes_state_lockguard", "hermes_state_lockowners", "hermes_state_maintenance",
	"hermes_state_messages", "hermes_state_portability", "hermes_state_profile_repair",
	"hermes_state_readpool", "hermes_state_registry", "hermes_state_repair", "hermes_state_rewind",
	"hermes_state_schema", "hermes_state_search", "hermes_state_sessions", "hermes_state_telegram",
	"hermes_state_titles", "hermes_state_usage", "hermes_state_user_copy", "hermes_state_wal",
	"hermes_time", "model_tools", "plugins", "providers", "registration_lifecycle", "run_agent",
	"tools", "toolsets", "tui_gateway", "utils",
}
