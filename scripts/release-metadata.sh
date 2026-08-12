#!/usr/bin/env bash
set -euo pipefail
[[ $# -ge 1 ]] || { echo "usage: $0 <artifact> [channel]" >&2; exit 2; }
artifact=$1
channel=${2:-dev}
[[ -f "$artifact" ]] || { echo "artifact not found" >&2; exit 1; }
version=$(tr -d '[:space:]' < VERSION)
bytes=$(stat -c '%s' "$artifact")
sha=$(sha256sum "$artifact" | awk '{print $1}')
commit=$(git rev-parse HEAD 2>/dev/null || echo unknown)
python3 - "$artifact" "$version" "$channel" "$bytes" "$sha" "$commit" <<'PY'
import json, os, sys
p,v,c,b,s,g=sys.argv[1:]
print(json.dumps({"product":"KINGAI OS","version":v,"channel":c,"artifact":os.path.basename(p),"size_bytes":int(b),"sha256":s,"source_commit":g},indent=2))
PY
