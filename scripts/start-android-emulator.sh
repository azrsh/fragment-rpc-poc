#!/usr/bin/env bash
set -euo pipefail
source "$(dirname "$0")/mobile-env.sh"
mkdir -p "$ANDROID_AVD_HOME"
if [ ! -f "$ANDROID_AVD_HOME/FragmentRPC.ini" ]; then
  printf 'no\n' | "$ANDROID_HOME/cmdline-tools/latest/bin/avdmanager" create avd \
    --name FragmentRPC --package 'system-images;android-35;google_apis;arm64-v8a' --device pixel_7
fi
if [ "${1:-}" = "--window" ]; then shift; else set -- -no-window "$@"; fi
exec "$ANDROID_HOME/emulator/emulator" -avd FragmentRPC -port 5556 \
  -no-audio -no-snapshot -no-boot-anim -gpu swiftshader_indirect -memory 2048 -cores 2 "$@"
