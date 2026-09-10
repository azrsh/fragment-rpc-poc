#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
bash scripts/generate-ios.sh
xcodegen generate --spec mobile/ios/project.yml
mkdir -p mobile/ios/FragmentRPC.xcodeproj/project.xcworkspace/xcshareddata/swiftpm
cp mobile/ios/Package.resolved mobile/ios/FragmentRPC.xcodeproj/project.xcworkspace/xcshareddata/swiftpm/Package.resolved
ios_device="${IOS_SIMULATOR:-iPhone 17}"
xcodebuild -project mobile/ios/FragmentRPC.xcodeproj -scheme FragmentRPC \
  -destination "platform=iOS Simulator,name=$ios_device" -derivedDataPath .local/ios-build build
ios_state="$(xcrun simctl list devices available -j | node -e 'let s="";process.stdin.on("data",d=>s+=d);process.stdin.on("end",()=>{let d=Object.values(JSON.parse(s).devices).flat().find(d=>d.name===process.argv[1]);if(!d)process.exit(1);console.log(d.state)})' "$ios_device")"
if [ "$ios_state" != Booted ]; then xcrun simctl boot "$ios_device"; fi
xcrun simctl bootstatus "$ios_device" -b
xcrun simctl install "$ios_device" .local/ios-build/Build/Products/Debug-iphonesimulator/FragmentRPC.app
xcrun simctl launch "$ios_device" dev.fragmentrpc.poc.ios
echo "Open Simulator to view Fragment RPC on $ios_device."
