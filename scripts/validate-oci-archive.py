#!/usr/bin/env python3
import argparse
import json
import tarfile
from pathlib import Path


OCI_INDEX = "application/vnd.oci.image.index.v1+json"
DOCKER_INDEX = "application/vnd.docker.distribution.manifest.list.v2+json"
OCI_MANIFEST = "application/vnd.oci.image.manifest.v1+json"
DOCKER_MANIFEST = "application/vnd.docker.distribution.manifest.v2+json"


def load_json_from_tar(tf: tarfile.TarFile, name: str):
    try:
        member = tf.getmember(name)
    except KeyError as exc:
        raise SystemExit(f"OCI archive is missing required object: {name}") from exc
    fh = tf.extractfile(member)
    if fh is None:
        raise SystemExit(f"unable to read {name}")
    try:
        return json.load(fh)
    except json.JSONDecodeError as exc:
        raise SystemExit(f"invalid JSON in OCI object {name}: {exc}") from exc


def blob_path_for_digest(digest: str) -> str:
    algorithm, separator, value = digest.partition(":")
    if separator != ":" or not algorithm or not value:
        raise SystemExit(f"invalid OCI digest: {digest!r}")
    if algorithm != "sha256":
        raise SystemExit(f"unsupported OCI digest algorithm: {algorithm}")
    return f"blobs/{algorithm}/{value}"


def descriptor_platform(descriptor: dict) -> str | None:
    platform = descriptor.get("platform") or {}
    os_name = platform.get("os")
    arch = platform.get("architecture")
    if os_name and arch:
        return f"{os_name}/{arch}"
    return None


def walk_descriptor(
    tf: tarfile.TarFile,
    descriptor: dict,
    found: set[str],
    visited: set[str],
) -> None:
    platform = descriptor_platform(descriptor)
    if platform:
        found.add(platform)

    digest = descriptor.get("digest")
    if not digest:
        raise SystemExit("OCI descriptor is missing digest")
    if digest in visited:
        return
    visited.add(digest)

    document = load_json_from_tar(tf, blob_path_for_digest(digest))
    if document.get("schemaVersion") != 2:
        raise SystemExit(f"OCI object {digest} has invalid schemaVersion")

    media_type = descriptor.get("mediaType") or document.get("mediaType") or ""
    manifests = document.get("manifests")

    # BuildKit may wrap a multi-platform image in one or more nested OCI indexes.
    # Platform metadata lives on the child descriptors, not necessarily on the
    # top-level descriptor in index.json, so indexes must be traversed recursively.
    if manifests is not None or media_type in {OCI_INDEX, DOCKER_INDEX}:
        if not isinstance(manifests, list):
            raise SystemExit(f"OCI index {digest} does not contain a manifests array")
        for child in manifests:
            if not isinstance(child, dict):
                raise SystemExit(f"OCI index {digest} contains an invalid descriptor")
            walk_descriptor(tf, child, found, visited)
        return

    if media_type and media_type not in {OCI_MANIFEST, DOCKER_MANIFEST}:
        raise SystemExit(f"unsupported OCI object mediaType {media_type!r} for {digest}")

    config = document.get("config")
    layers = document.get("layers")
    if not isinstance(config, dict) or not isinstance(layers, list):
        raise SystemExit(f"OCI image manifest {digest} is missing config or layers")

    # Assert referenced config/layer blobs are present. This makes the validator
    # prove archive integrity beyond merely finding architecture labels.
    referenced = [config, *layers]
    for item in referenced:
        item_digest = item.get("digest") if isinstance(item, dict) else None
        if not item_digest:
            raise SystemExit(f"OCI image manifest {digest} contains a descriptor without digest")
        try:
            tf.getmember(blob_path_for_digest(item_digest))
        except KeyError as exc:
            raise SystemExit(
                f"OCI image manifest {digest} references missing blob {item_digest}"
            ) from exc


def main() -> int:
    parser = argparse.ArgumentParser(description="Validate a KINGAI OS multi-architecture OCI image archive")
    parser.add_argument("archive", type=Path)
    parser.add_argument(
        "--platform",
        action="append",
        dest="platforms",
        default=[],
        help="required platform in os/arch form; may be repeated",
    )
    args = parser.parse_args()

    required = set(args.platforms or ["linux/amd64", "linux/arm64"])
    if not args.archive.is_file():
        raise SystemExit(f"OCI archive not found: {args.archive}")

    with tarfile.open(args.archive, "r:*") as tf:
        index = load_json_from_tar(tf, "index.json")
        if index.get("schemaVersion") != 2:
            raise SystemExit("OCI index schemaVersion must be 2")
        manifests = index.get("manifests")
        if not isinstance(manifests, list) or not manifests:
            raise SystemExit("OCI index must contain at least one manifest descriptor")

        found: set[str] = set()
        visited: set[str] = set()
        for descriptor in manifests:
            if not isinstance(descriptor, dict):
                raise SystemExit("OCI index contains an invalid descriptor")
            walk_descriptor(tf, descriptor, found, visited)

        missing = required - found
        if missing:
            raise SystemExit(
                "OCI archive missing required platform(s): " + ", ".join(sorted(missing))
            )

    print("KINGAI OS OCI archive validated: " + ", ".join(sorted(found)))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
