#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
npm run generate
mkdir -p mobile/ios/Generated
cmp -s generated/NativeFragments.swift mobile/ios/Generated/NativeFragments.swift || cp generated/NativeFragments.swift mobile/ios/Generated/NativeFragments.swift
protoc -I generated \
  --plugin=protoc-gen-swift=.local/mobile-tools/swift-protobuf/.build/release/protoc-gen-swift \
  --plugin=protoc-gen-connect-swift=.local/mobile-tools/connect-swift/.build/release/protoc-gen-connect-swift \
  --swift_out=mobile/ios/Generated --swift_opt=Visibility=Public \
  --connect-swift_out=mobile/ios/Generated \
  --connect-swift_opt=GenerateAsyncMethods=true,GenerateCallbackMethods=false,Visibility=Public \
  app.proto
