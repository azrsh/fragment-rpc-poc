#!/usr/bin/env bash
mobile_repo="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
export JAVA_HOME="${JAVA_HOME:-$mobile_repo/.local/mobile-tools/jdk/Contents/Home}"
export ANDROID_HOME="${ANDROID_HOME:-$mobile_repo/.local/mobile-tools/android-sdk}"
export ANDROID_USER_HOME="$mobile_repo/.local/mobile-tools/android-user"
export ANDROID_AVD_HOME="$mobile_repo/.local/mobile-tools/avd"
export GRADLE_USER_HOME="$mobile_repo/.local/mobile-tools/gradle-cache"
export PATH="$JAVA_HOME/bin:$ANDROID_HOME/platform-tools:$ANDROID_HOME/emulator:$mobile_repo/.local/mobile-tools/gradle-8.11.1/bin:$PATH"
