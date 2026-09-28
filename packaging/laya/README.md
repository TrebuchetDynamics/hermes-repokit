# Pinned local Laya image

This recipe packages the qualified upstream Nerve Laya service with its CPU
dependencies and model. It is a packaging milestone: RepoKit can generate the
sidecar, but does not yet install or enable Nerve on managed profiles.

From the repository root, build for the qualified Linux amd64 platform:

```sh
docker --context default build --platform linux/amd64 \
  --iidfile /tmp/repokit-laya.iid packaging/laya
REPOKIT_LAYA_IMAGE="$(cat /tmp/repokit-laya.iid)" \
  go test ./internal/supervision -run TestLayaImageSelfContained -count=1 -v
```

The build downloads only pinned dependencies, the checksum-verified upstream
archive and the five checksum-verified model files. The final image needs no
downloads or host dependency/model mounts. The test loads that image with
networking disabled and checks dependency versions and every model hash.

Select the resulting full local image ID in the target repository:

```sh
hermes-repokit plan --laya-image "$(cat /tmp/repokit-laya.iid)"
hermes-repokit install --laya-image "$(cat /tmp/repokit-laya.iid)"
```

Use the same Docker context for build and target installation. Mutable tags,
missing local images and mismatched recipe/platform metadata are refused.
An existing Hermes-only deployment must first run ordinary `install` to add the
OpenViking scaffold; Laya then upgrades that exact generated Compose, preserving
its prior bytes in `.hermes/compose.before-laya.yaml`. Owner edits are preserved
and rejected for automatic adoption. Reruns retain the selected Laya image ID.

Run the coordinated Hermes/Laya recreation command printed by `install`.
Laya shares Hermes's network namespace and serves only loopback port 8765.
Recreate both services together if that namespace changes. No port is published,
and the service has a read-only filesystem with temporary scratch space.

No registry publication or image portability is implied by a local image ID.
Other Docker daemons need their own build and explicit selection. Image labels
identify the recipe; they are not a cryptographic publisher attestation.

Native Nerve admission/configuration/activation, all-role supervision, dynamic
specialist provisioning, coordinated recreation acceptance and full-stack
dogfood remain pending. `verify` continues to report supervision as unknown.

Qualification on 2026-09-27: the built image passed the content test and actual
upstream health plus choice/score/noul inference with networking disabled,
nonroot UID, read-only filesystem, four CPUs, 6 GiB memory and no host mounts.
