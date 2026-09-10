#!/usr/bin/env bash
set -euo pipefail
source "$(dirname "$0")/mobile-env.sh"
cd "$mobile_repo"
npm run generate
bash scripts/android-gradle.sh :app:assembleDebug
adb -s "${ANDROID_SERIAL:-emulator-5556}" install -r mobile/android/app/build/outputs/apk/debug/app-debug.apk
adb -s "${ANDROID_SERIAL:-emulator-5556}" shell am start -n dev.fragmentrpc.poc.android/dev.fragmentrpc.MainActivity
