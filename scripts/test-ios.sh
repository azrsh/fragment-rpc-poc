#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
bash scripts/generate-ios.sh
xcodegen generate --spec mobile/ios/project.yml
mkdir -p mobile/ios/FragmentRPC.xcodeproj/project.xcworkspace/xcshareddata/swiftpm
cp mobile/ios/Package.resolved mobile/ios/FragmentRPC.xcodeproj/project.xcworkspace/xcshareddata/swiftpm/Package.resolved
exec xcodebuild -project mobile/ios/FragmentRPC.xcodeproj -scheme FragmentRPC \
  -destination "platform=iOS Simulator,name=${IOS_SIMULATOR:-iPhone 17}" \
  -derivedDataPath .local/ios-build -parallel-testing-enabled NO test
