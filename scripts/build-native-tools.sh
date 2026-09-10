#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
bash scripts/build-swift-extractor.sh
bash scripts/android-gradle.sh :graphql-tools:installDist >&2
