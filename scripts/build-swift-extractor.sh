#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
env -u SDKROOT swift build --package-path mobile/swift-graphql --product extract-graphql >&2
