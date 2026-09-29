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
