#!/usr/bin/env bash
set -euo pipefail

DEST="${KINGAI_SENTINEL_FEED_DIR:-/var/lib/kingai/sentinel/feeds}"
TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT
mkdir -p "$DEST"

fetch() {
  local url="$1" out="$2"
  curl --fail --silent --show-error --location \
    --proto '=https' --tlsv1.2 \
    --connect-timeout 10 --max-time 120 \
    --retry 2 --retry-delay 2 \
    "$url" -o "$TMP/$out"
  test -s "$TMP/$out"
}

fetch "https://www.cisa.gov/sites/default/files/feeds/known_exploited_vulnerabilities.json" "cisa-kev.json"
fetch "https://epss.empiricalsecurity.com/epss_scores-current.csv.gz" "first-epss.csv.gz"
fetch "https://services.nvd.nist.gov/rest/json/cves/2.0?hasKev&resultsPerPage=2000" "nvd-kev.json"

python3 - "$TMP/cisa-kev.json" "$TMP/nvd-kev.json" <<'PY'
import json, sys
for path in sys.argv[1:]:
    with open(path, 'rb') as f:
        json.load(f)
PY

gzip -t "$TMP/first-epss.csv.gz"

for f in cisa-kev.json first-epss.csv.gz nvd-kev.json; do
  install -m 0640 "$TMP/$f" "$DEST/$f.new"
  mv -f "$DEST/$f.new" "$DEST/$f"
done

{
  printf 'synced_at=%s\n' "$(date -u +%Y-%m-%dT%H:%M:%SZ)"
  (cd "$DEST" && sha256sum cisa-kev.json first-epss.csv.gz nvd-kev.json)
} > "$DEST/manifest.txt.new"
mv -f "$DEST/manifest.txt.new" "$DEST/manifest.txt"

echo "Sentinel intelligence sync complete"
