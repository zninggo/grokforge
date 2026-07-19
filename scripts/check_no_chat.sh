#!/usr/bin/env bash
# Fail if OpenAI/Anthropic-style chat gateway paths appear in product code.
# Note: YYDS mail uses /v1/messages/next — that is allowed.
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

search() {
  local pat="$1"
  if command -v rg >/dev/null 2>&1; then
    rg -n --hidden \
      -g '!.git/**' -g '!docs/**' -g '!frontend/node_modules/**' \
      -g '!frontend/.next/**' -g '!frontend/out/**' \
      -g '!backend/bin/**' -g '!backend/internal/web/dist/**' \
      -g '!scripts/check_no_chat.sh' \
      -g '*.go' -g '*.ts' -g '*.tsx' \
      -F "$pat" . || return 1
    return 0
  fi
  grep -RIn --include='*.go' --include='*.ts' --include='*.tsx' \
    --exclude-dir=.git --exclude-dir=docs --exclude-dir=node_modules \
    --exclude-dir=.next --exclude-dir=out --exclude-dir=bin --exclude-dir=dist \
    --exclude='check_no_chat.sh' \
    -F "$pat" . || return 1
  return 0
}

PATTERNS=(
  '/v1/chat/completions'
  'chat/completions'
  '/v1/responses'
  'ChatCompletion'
  'chat.completions'
)

hits=0
for p in "${PATTERNS[@]}"; do
  if search "$p"; then
    echo "FORBIDDEN pattern found: $p" >&2
    hits=$((hits + 1))
  fi
done

if [[ "$hits" -gt 0 ]]; then
  echo "check_no_chat: FAILED ($hits patterns)" >&2
  exit 1
fi
echo "check_no_chat: OK"
