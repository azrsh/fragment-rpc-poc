#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
mkdir -p .local/mobile-tools
if [ ! -d .local/mobile-tools/swift-protobuf ]; then
  git clone --depth 1 --branch 1.31.0 https://github.com/apple/swift-protobuf.git .local/mobile-tools/swift-protobuf
fi
if [ ! -d .local/mobile-tools/connect-swift ]; then
  git clone --depth 1 --branch 1.2.3 https://github.com/connectrpc/connect-swift.git .local/mobile-tools/connect-swift
fi
swift build --package-path .local/mobile-tools/swift-protobuf -c release --product protoc-gen-swift
swift build --package-path .local/mobile-tools/connect-swift -c release --product protoc-gen-connect-swift
