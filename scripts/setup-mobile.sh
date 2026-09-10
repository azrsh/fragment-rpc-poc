#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
mobile_tools="$PWD/.local/mobile-tools"
mkdir -p "$mobile_tools/downloads" "$mobile_tools/android-sdk/cmdline-tools" "$mobile_tools/jdk"

if [ ! -x "$mobile_tools/jdk/Contents/Home/bin/java" ]; then
  curl -fL --retry 3 'https://api.adoptium.net/v3/binary/latest/17/ga/mac/aarch64/jdk/hotspot/normal/eclipse' -o "$mobile_tools/downloads/jdk17.tar.gz"
  tar -xzf "$mobile_tools/downloads/jdk17.tar.gz" --strip-components=1 -C "$mobile_tools/jdk"
fi
export JAVA_HOME="$mobile_tools/jdk/Contents/Home"
export ANDROID_HOME="$mobile_tools/android-sdk"
export ANDROID_USER_HOME="$mobile_tools/android-user"
export PATH="$JAVA_HOME/bin:$PATH"

if [ ! -x "$ANDROID_HOME/cmdline-tools/latest/bin/sdkmanager" ]; then
  curl -fL --retry 3 'https://dl.google.com/android/repository/commandlinetools-mac_arm64-15859902_latest.zip' -o "$mobile_tools/downloads/android-cli.zip"
  printf '%s  %s\n' '835b62a26162b229b441d1f6d4680383815a270809eb33522c0d480fa5002c4e' "$mobile_tools/downloads/android-cli.zip" | shasum -a 256 -c -
  unzip -q "$mobile_tools/downloads/android-cli.zip" -d "$ANDROID_HOME/cmdline-tools"
  mv "$ANDROID_HOME/cmdline-tools/cmdline-tools" "$ANDROID_HOME/cmdline-tools/latest"
fi
if [ ! -x "$mobile_tools/gradle-8.11.1/bin/gradle" ]; then
  curl -fL --retry 3 'https://services.gradle.org/distributions/gradle-8.11.1-bin.zip' -o "$mobile_tools/downloads/gradle.zip"
  curl -fL --retry 3 'https://services.gradle.org/distributions/gradle-8.11.1-bin.zip.sha256' -o "$mobile_tools/downloads/gradle.sha256"
  printf '%s  %s\n' "$(cat "$mobile_tools/downloads/gradle.sha256")" "$mobile_tools/downloads/gradle.zip" | shasum -a 256 -c -
  unzip -q "$mobile_tools/downloads/gradle.zip" -d "$mobile_tools"
fi
if [ "${1:-}" = "--accept-android-licenses" ]; then
  set +o pipefail
  yes | "$ANDROID_HOME/cmdline-tools/latest/bin/sdkmanager" --sdk_root="$ANDROID_HOME" --licenses
  license_status=${PIPESTATUS[1]}
  set -o pipefail
  test "$license_status" -eq 0
fi
"$ANDROID_HOME/cmdline-tools/latest/bin/sdkmanager" --sdk_root="$ANDROID_HOME" \
  'platform-tools' 'platforms;android-35' 'build-tools;35.0.0' 'emulator' 'system-images;android-35;google_apis;arm64-v8a'
echo "Android tooling installed under $mobile_tools"
