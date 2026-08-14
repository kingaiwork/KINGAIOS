#!/usr/bin/env python3
import argparse
import json
import tarfile
from pathlib import Path


def load_json_from_tar(tf: tarfile.TarFile, name: str):
    member = tf.getmember(name)
    fh = tf.extractfile(member)
    if fh is None:
        raise RuntimeError(f"unable to read {name}")
    return json.load(fh)


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

        found = set()
        for descriptor in index.get("manifests", []):
            platform = descriptor.get("platform") or {}
            os_name = platform.get("os")
            arch = platform.get("architecture")
            if os_name and arch:
                found.add(f"{os_name}/{arch}")

            digest = descriptor.get("digest", "")
            if digest.startswith("sha256:"):
                blob_path = "blobs/sha256/" + digest.split(":", 1)[1]
                try:
                    manifest = load_json_from_tar(tf, blob_path)
                except KeyError:
                    raise SystemExit(f"missing manifest blob for {digest}")
                if manifest.get("schemaVersion") != 2:
                    raise SystemExit(f"manifest {digest} has invalid schemaVersion")

        missing = required - found
        if missing:
            raise SystemExit(
                "OCI archive missing required platform(s): " + ", ".join(sorted(missing))
            )

    print("KINGAI OS OCI archive validated: " + ", ".join(sorted(found)))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
