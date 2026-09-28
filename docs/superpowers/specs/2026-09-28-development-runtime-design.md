# Repository development runtime

The owner requests full repository development from every configured interactive
channel. Hermes remains the native six-profile organization. RepoKit generates
ordinary Compose and build recipes; no Pi, mandatory Codex, host toolchain or
RepoKit runtime callback is introduced.

The default Hermes service builds a derived image from the exact qualified
Hermes digest. Its inherited tools plus checksum-pinned additions supply Git,
Bash, CA certificates, curl, jq, ripgrep, Python, Node/npm and build utilities.
Root manifests select the small supported toolchain matrix; Go is included for
go.mod. Rust/JVM and unsupported version constraints are reported, not silently
called ready. No repository package scripts execute during detection or image
construction. Runtime tools are baked into the image, not installed per start.

Keep /workspace read/write and /opt/data private state. Native terminal backend
is local with cwd /workspace. Executor receives native file, terminal,
code_execution, skills and memory; task lifecycle remains Hermes-owned. Reviewer
can run checks but remains non-writing by policy, not a claimed OS sandbox.

Docker acceptance is an explicit install selection and Compose profile. A
separate pinned privileged Docker-in-Docker daemon owns only its test storage,
socket and scratch volumes. It receives no host Docker socket, repository bind
or Hermes private state. Scratch paths must match in client and daemon: Docker
bind sources are resolved on the daemon. No public TCP daemon is exposed.
Privilege remains a host-kernel trust boundary; this is not VM-grade isolation.
Enabling this service is deliberate authority for Docker tests, not a fix for
Codex sandbox restrictions. Normal coding works without it.

Verification stays observational: exact generated build recipe, selected image
identity, standard version commands, project toolchain requirements, workdir and
read/write mount metadata. Tool readiness and optional Docker-test daemon health
are independent from authenticated chat, memory recall and real reviewed work.

Preserve owner source/runtime state. Upgrade only exact recognized generated
Compose and recipes, retaining old Compose. Refuse edited build inputs rather
than overwriting. Source fixtures must cover fresh install, repeat install,
legacy upgrade, unsafe manifests, tool deficits and optional daemon isolation.
Actual image builds and Telegram/model acceptance remain pending until required
host socket/Docker gates and private setup succeed. Never weaken those gates.
