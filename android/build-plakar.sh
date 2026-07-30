#!/bin/sh
#
# Cross-compile the plakar CLI for Android and drop it where the APK build
# expects it. Must run before `gradle assembleDebug`.
#
# The binary is named libplakar.so because only files matching lib*.so in the
# jniLibs tree get extracted to the app's native library directory, and that
# directory is the only place Android 10+ allows an app to exec from.

set -eu

cd "$(dirname "$0")"

OUT=app/src/main/jniLibs/arm64-v8a/libplakar.so

mkdir -p "$(dirname "$OUT")"

# CGO_ENABLED=0 keeps this buildable without the NDK. The tradeoff is Go's
# pure-Go resolver, which looks for /etc/resolv.conf -- a file Android does not
# have -- so hostnames will not resolve. Address the store by IP, or rebuild
# with the NDK and CGO_ENABLED=1 to get bionic's resolver.
CGO_ENABLED=0 GOOS=android GOARCH=arm64 go build -ldflags="-s -w" -o "$OUT" ..

echo "built $OUT ($(du -h "$OUT" | cut -f1))"
