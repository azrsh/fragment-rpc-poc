#!/usr/bin/env bash
set -euo pipefail
source "$(dirname "$0")/mobile-env.sh"
exec "$mobile_repo/.local/mobile-tools/gradle-8.11.1/bin/gradle" -p "$mobile_repo/mobile/android" --console=plain "$@"
