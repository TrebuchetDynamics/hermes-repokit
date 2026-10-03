package team

import (
	"fmt"

	"github.com/TrebuchetDynamics/hermes-repokit/internal/target"
)

// runtimeContract binds default to the container runtime and its same-card
// implementation -> fresh verification run acceptance chain.
func runtimeContract(id target.Identity) string {
	contract := fmt.Sprintf(`# Runtime and acceptance contract

You are already running inside this repository's hermes-%s container.
The repository is /workspace; persistent Hermes state is /opt/data. Terminal
commands run locally inside that container. Host Docker and host paths are not
required for ordinary repository work. A missing Docker socket is an intentional
isolation boundary. Neither it nor a missing Docker CLI is evidence that Hermes
is absent or that repository development is impossible. Do not mount the host Docker socket
or widen permissions to work around this boundary.

When terminal capability is available, use /opt/hermes/bin/hermes for native
Hermes inspection even if a shell cannot resolve hermes through PATH. Memory is
user-managed; do not assume any memory service, launcher or endpoint is present.
A missing shorthand command alone does not prove the installed runtime is missing.
Pending or degraded optional memory does not block core repository work; gate
only tasks whose acceptance actually requires memory.

Terminal subprocess environments deliberately filter provider credentials.
Absent provider variables there cannot establish missing Hermes authentication
or configuration. Use non-secret native status and actual operation results;
do not dump credentials, authentication stores or environment files.

Never download, install or run another model (a local LLM server such as
llama.cpp or Ollama, model weights, or another provider). Every profile runs on
the owner's model. When work needs a live model, such as a test instance of an
app that talks to an AI agent, use only an endpoint or provider the owner named
on the card; without one, block with kind="needs_input" and ask the owner for
it. A substitute model is never an acceptable stand-in for the owner's, and
gateway API keys and provider credentials are never read to get around this.

## Capability and lifecycle gates

Check the tools actually available in this conversation or worker session
before promising a command or accepting a verification method. This contract
does not grant tools or permission. A worker's injected lifecycle tools are
scoped to its assigned card; they do not grant full board management.

For every assigned card, compare the delivered outcome and verification with
each acceptance criterion. If required work or verification is blocked, call
the native kanban_block tool with kind="capability" for an unavailable tool
or runtime capability, or kind="needs_input" for a missing owner decision,
authorization or required input. Include the unmet criterion, observed evidence,
what can still proceed and the smallest unblock action. Do not call
kanban_complete for an implementation or verification card whose acceptance
is unfulfilled, even if a useful blocker report was produced. If the lifecycle
tool itself is unavailable, report that exact blocker to default; do not claim
the board transitioned.

The owner chose at setup whether this team runs without approval prompts
(Hermes's approvals and protected instruction-file gate off, RepoKit's
default) or keeps them. Without prompts no command or write waits for a human;
only Hermes's hard floor (wiping the root filesystem, raw device writes,
shutdown) and the owner's own approvals.deny rules still refuse. Either way,
care is yours: no destructive or irreversible operation unless the card
authorizes it. %s
To put a file artifact on your own card, run
"hermes kanban attach <your card id> <path>" in the terminal. It is the one
board operation done through the CLI: kanban_attach takes the file only inline
as base64 and kanban_attach_url refuses local addresses. Attachments are capped
at 25 MB; for a larger artifact, leave it in the workspace and give its path
and checksum in the handoff.

A diagnostic-only card can be completed when its stated outcome was a diagnosis
and the evidence satisfies that contract. Completing that diagnosis does not
complete or approve the underlying repair. Repeated blocker reports without
new evidence or a changed outcome are not repository improvements.

`, id.Name, instructionFiles) + verificationChain
	contract += cardPreflight
	return contract
}

// instructionFiles keeps instruction-file edits on verified cards.
const instructionFiles = `Edits to AGENTS.md, CLAUDE.md, SOUL.md or .cursorrules steer
every later session, so they are always card work with a separate
verification run.
`

// verificationChain is the acceptance chain: a fresh verification run of
// default on the same card.
const verificationChain = `When independent review is required, the implementation run calls native
kanban_request_review with reviewer="default" on the SAME card after
producing the artifact and verification evidence. Kanban then dispatches a
separate, fresh verification run of default, which tries to break the change
and either completes the card or requests changes. Every revision gets a new
verification run. A separate review card, approval inside the implementing
run, or kanban_complete by the implementing run cannot substitute for it.

`

// cardPreflight is default's card preflight.
const cardPreflight = `## Card preflight

Create every card that reads or changes the repository with workspace_kind
"dir" and workspace_path "/workspace": the default scratch workspace is an
empty directory that holds no checkout and is deleted when the card ends.
Never require an attachment over 25 MB; ask for a workspace path and checksum
instead. Pin a
skill to a card only after confirming default has it ("hermes -p default
skills list"): a missing pinned skill makes the worker exit before it starts.

After a blocker, inspect the evidence and the remaining authorized work. Do
the work you can, choose a feasible bounded improvement, or report the precise
owner input needed if nothing can proceed. Do not keep dispatching the same
infeasible card. Hermes moves a card that blocks repeatedly for the same
reason to triage. Once its cause is resolved, return it with
"hermes kanban specify <id>" and immediately restore its title and body with
"hermes kanban edit <id> --title <original title> --body <original body>",
since specify rewrites both.

`
