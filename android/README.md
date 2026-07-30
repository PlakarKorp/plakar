# plakar Android PoC

A minimal, unrooted Android app that runs the plakar CLI against `/sdcard`.

This is a proof of concept, not a product: one activity, one button, no
foreground service, no scheduling. It exists to answer whether the existing Go
codebase can back up a tablet without root. It can.

## How it works

plakar is cross-compiled for `android/arm64` and shipped inside the APK as
`libplakar.so`. The app execs it with `ProcessBuilder` and streams its output
into a text view.

Three details make that work, and each one is a trap if you get it wrong:

- **`useLegacyPackaging = true`** (`app/build.gradle.kts`). Android 10+ enforces
  W^X on app-writable storage, so the extracted native library directory is the
  only place an app may exec from. Modern packaging keeps libraries inside the
  APK and maps them from there, so nothing lands on disk and there is no path to
  exec; legacy packaging extracts at install time and gives us a real file.
- **The `lib*.so` name.** Only files matching that pattern are extracted into
  the native library directory. `plakar` alone would be packaged but never
  unpacked.
- **`HOME` and the XDG variables.** plakar resolves its config, cache and data
  directories from `XDG_CONFIG_HOME`/`XDG_CACHE_HOME`/`XDG_DATA_HOME`, falling
  back to `$HOME`. Android sets none of them, and Go's `os/user` needs `$HOME`
  to return a usable current user at all, so the app points them at its private
  directory before exec.

## Building

Requires the Android SDK (platform 35, build-tools) and a Go toolchain.

```sh
./build-plakar.sh          # cross-compiles plakar into app/src/main/jniLibs/
gradle assembleDebug       # -> app/build/outputs/apk/debug/app-debug.apk
adb install -r app/build/outputs/apk/debug/app-debug.apk
```

CI builds this on every push touching `android/` and uploads the APK as a
workflow artifact, so you do not need a local SDK to get an installable build.

## Using it

1. Create the store and serve it from a machine on the same network:
   `plakar create && plakar server -listen 0.0.0.0:9876`
2. Launch the app, tap **Grant all files access**, and enable the toggle.
3. Tap **plakar version** as a smoke test that the binary runs at all.
4. Enter the store location and passphrase, tap **Back up /sdcard**.

Keep the screen on. There is no foreground service, so Android will freeze the
process once the app goes to the background.

## What it does and does not capture

Covered: everything under `/sdcard` — DCIM, Pictures, Documents, Downloads.

Not covered, and not fixable at this level:

- `Android/data` and `Android/obb`, which All files access explicitly excludes.
  Only the shell user reaches those, via adb or Shizuku.
- Other apps' private data under `/data/data`. No app-level API exists, at any
  permission. That needs root, or Seedvault on a ROM that ships it.

## Known limitations of the PoC

- **Hostnames will not resolve.** `build-plakar.sh` uses `CGO_ENABLED=0` so it
  builds without the NDK, which means Go's pure-Go DNS resolver and a lookup for
  `/etc/resolv.conf` that does not exist on Android. Address the store by IP.
  Rebuilding with the NDK and `CGO_ENABLED=1` gets bionic's resolver and fixes
  it.
- **No foreground service**, so a long backup needs the screen awake. Doing this
  properly means a user-initiated data transfer job — Android 15 caps `dataSync`
  foreground services at six hours per day, which a first full backup will
  exceed.
- **arm64 only.** `android/amd64` and `android/arm` require cgo, so they need
  the NDK.
- `ACCESS_MEDIA_LOCATION` is declared, which is what stops the platform
  redacting GPS tags out of photos. Worth verifying on your own device by
  comparing EXIF from a backed-up photo against the original.
