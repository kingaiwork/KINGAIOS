#!/usr/bin/env bash
set -euo pipefail

ROOT=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

cat > "$tmp/qml6" <<'EOF'
#!/usr/bin/env bash
printf '%s\n' "$@" > "$KINGAI_TEST_CAPTURE"
EOF
chmod +x "$tmp/qml6"
export PATH="$tmp:$PATH"
export KINGAI_TEST_CAPTURE="$tmp/args"

run_case() {
  local expected="$1"
  shift
  : > "$KINGAI_TEST_CAPTURE"
  bash "$ROOT/desktop/intelligence/launch.sh" "$@"
  mapfile -t args < "$KINGAI_TEST_CAPTURE"
  [[ "${args[0]:-}" == "/usr/share/kingai/desktop/intelligence/Main.qml" ]]
  [[ "${args[1]:-}" == "--" ]]
  [[ "${args[2]:-}" == "--center" ]]
  [[ "${args[3]:-}" == "$expected" ]] || {
    printf 'expected center %s, got %s\n' "$expected" "${args[3]:-<missing>}" >&2
    exit 1
  }
}

run_case home
run_case agents --center agents
run_case approvals kingai://approvals
run_case tasks kingai://tasks/details
run_case health health
run_case home kingai://unknown
run_case home ../../etc/passwd

echo "KINGAI Intelligence launcher tests: OK"
