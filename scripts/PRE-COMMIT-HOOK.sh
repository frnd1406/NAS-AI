#!/bin/bash
# Secret Detection Pre-Commit Hook
# Created: 2025-11-14 - Secrets Rotation Prevention
#
# Installation:
#   cp docs/PRE-COMMIT-HOOK.sh .git/hooks/pre-commit
#   chmod +x .git/hooks/pre-commit

echo "🔍 Scanning for secrets..."

# Check for Vault tokens
if git diff --cached | grep -E "hvs\.[a-zA-Z0-9]{20,}"; then
    echo "❌ ERROR: Vault token detected in staged files!"
    echo "Please remove the token before committing."
    exit 1
fi

# Check for Resend API keys
if git diff --cached | grep -E "re_[a-zA-Z0-9]{20,}"; then
    echo "❌ ERROR: Resend API key detected in staged files!"
    echo "Please remove the API key before committing."
    exit 1
fi

# Check for generic API keys
if git diff --cached | grep -E "api[_-]?key\s*[=:]\s*[\"'][^\"']{20,}[\"']"; then
    echo "❌ ERROR: Potential API key detected in staged files!"
    echo "Please review and remove if it's a real secret."
    exit 1
fi

echo "✅ No secrets detected. Commit allowed."
exit 0
