# Pinned local Laya image

This recipe packages the qualified upstream Nerve Laya service with its CPU
dependencies and model. Ordinary `hermes-repokit install` generates all three
services: Hermes, OpenViking, and Laya. It copies this exact embedded build recipe
to `.hermes/laya-image`; the printed ordinary Docker Compose command builds Laya
without needing the RepoKit checkout or executable afterward.

The default service keeps writable Hugging Face and Torch caches in the private
repository-local `.hermes/laya` directory. Checksum-verified model weights persist
in the image at `/model`; runtime model downloads are disabled. The model remains
available after container recreation, and writable caches survive in the bind
mount. OpenViking still needs private native model/provider setup.

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
Ordinary installation upgrades an exact generated Hermes-only or
Hermes/OpenViking deployment to the default three-service stack. It preserves
prior Compose bytes in `.hermes/compose.hermes-only.yaml` or
`.hermes/compose.before-laya.yaml`, respectively. Owner Compose and recipe edits
are preserved and rejected for automatic adoption. Native configuration and
credentials stay unchanged. Reruns retain an explicitly selected Laya image ID.

Run the coordinated Hermes/Laya recreation command printed by `install`.
Laya shares Hermes's network namespace and serves only loopback port 8765.
Recreate both services together if that namespace changes. No port is published,
and the service has a read-only filesystem with temporary scratch space.

No registry publication or image portability is implied by a local image ID.
Other Docker daemons need their own build and explicit selection. Image labels
identify the recipe; they are not a cryptographic publisher attestation.

Run native `setup` after starting the services to configure profiles and their
Nerve supervision. A generated sidecar alone does not establish that native
supervision is active; `verify` reports the observed runtime and profile state.

Qualification on 2026-09-27: the built image passed the content test and actual
upstream health plus choice/score/noul inference with networking disabled,
nonroot UID, read-only filesystem, four CPUs, 6 GiB memory and no host mounts.
