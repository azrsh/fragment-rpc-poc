#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
npm run generate
exec bash scripts/android-gradle.sh :app:connectedDebugAndroidTest
