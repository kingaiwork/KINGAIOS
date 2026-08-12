#!/usr/bin/env python3
import hashlib
import json
import os
import re
import sys
from datetime import datetime, timezone
from pathlib import Path


def fail(msg: str) -> None:
    raise SystemExit(msg)


def spdx_id(value: str) -> str:
    value = re.sub(r"[^A-Za-z0-9.-]+", "-", value).strip("-.")
    return value or "package"


def created_time() -> str:
    epoch = int(os.environ.get("SOURCE_DATE_EPOCH", "0"))
    return datetime.fromtimestamp(epoch, tz=timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ")


def main() -> None:
    if len(sys.argv) != 6:
        fail("usage: generate-sbom.py <packages.tsv> <output.spdx.json> <version> <profile> <arch>")

    package_file, output_file, version, profile, arch = sys.argv[1:]
    package_path = Path(package_file)
    if not package_path.is_file():
        fail(f"package inventory not found: {package_path}")

    rows = []
    for line in package_path.read_text(encoding="utf-8").splitlines():
        if not line.strip():
            continue
        parts = line.split("\t")
        if len(parts) != 3:
            fail(f"invalid package inventory row: {line!r}")
        rows.append(tuple(parts))

    seed = package_path.read_bytes() + b"\0" + version.encode() + b"\0" + profile.encode() + b"\0" + arch.encode()
    digest = hashlib.sha256(seed).hexdigest()
    namespace = f"https://os.kingai.work/spdx/{version}/{profile}/{arch}/{digest}"
    root_id = "SPDXRef-Package-KINGAI-OS"

    packages = [{
        "name": "KINGAI OS",
        "SPDXID": root_id,
        "versionInfo": version,
        "downloadLocation": "NOASSERTION",
        "filesAnalyzed": False,
        "licenseConcluded": "NOASSERTION",
        "licenseDeclared": "NOASSERTION",
        "copyrightText": "NOASSERTION",
        "summary": f"KINGAI OS {profile} image package inventory for {arch}. Third-party packages retain their upstream terms."
    }]
    relationships = []

    used_ids = set()
    for index, (name, pkg_version, pkg_arch) in enumerate(rows, start=1):
        base = f"SPDXRef-Package-{spdx_id(name)}-{index}"
        spdx = base
        while spdx in used_ids:
            index += 1
            spdx = f"{base}-{index}"
        used_ids.add(spdx)
        packages.append({
            "name": name,
            "SPDXID": spdx,
            "versionInfo": pkg_version,
            "downloadLocation": "NOASSERTION",
            "filesAnalyzed": False,
            "licenseConcluded": "NOASSERTION",
            "licenseDeclared": "NOASSERTION",
            "copyrightText": "NOASSERTION",
            "externalRefs": [{
                "referenceCategory": "PACKAGE-MANAGER",
                "referenceType": "purl",
                "referenceLocator": f"pkg:deb/ubuntu/{name}@{pkg_version}?arch={pkg_arch}"
            }]
        })
        relationships.append({
            "spdxElementId": root_id,
            "relationshipType": "CONTAINS",
            "relatedSpdxElement": spdx
        })

    document = {
        "spdxVersion": "SPDX-2.3",
        "dataLicense": "CC0-1.0",
        "SPDXID": "SPDXRef-DOCUMENT",
        "name": f"KINGAI-OS-{profile}-{version}-{arch}",
        "documentNamespace": namespace,
        "creationInfo": {
            "created": created_time(),
            "creators": ["Organization: KINGAI", "Tool: KINGAI OS deterministic package SBOM generator 0.1"]
        },
        "documentDescribes": [root_id],
        "packages": packages,
        "relationships": relationships,
        "comment": "Baseline package SBOM. NOASSERTION is intentional where license conclusions have not yet been machine-verified; release license compliance is tracked separately from this inventory."
    }

    out = Path(output_file)
    out.parent.mkdir(parents=True, exist_ok=True)
    tmp = out.with_suffix(out.suffix + ".tmp")
    tmp.write_text(json.dumps(document, indent=2, sort_keys=True) + "\n", encoding="utf-8")
    tmp.replace(out)


if __name__ == "__main__":
    main()
