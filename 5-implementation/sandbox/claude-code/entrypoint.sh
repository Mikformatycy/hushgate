#!/bin/sh
# Skip first-run onboarding and trust the demo workspace, so `claude` starts
# straight into a session. Login comes from CLAUDE_CODE_OAUTH_TOKEN.
set -e
if [ ! -f "$HOME/.claude.json" ]; then
  cat > "$HOME/.claude.json" <<JSON
{
  "hasCompletedOnboarding": true,
  "projects": { "/workspace": { "hasTrustDialogAccepted": true } }
}
JSON
fi
exec "$@"
