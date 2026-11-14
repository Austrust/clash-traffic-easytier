# Linux EasyTier build

This fork preserves the EasyTier discovery and authenticated incremental statistics sync added to Clash Traffic 0.1.4. Upstream: https://github.com/lufficc/clash-traffic (base commit `1c2524923ab14de59068e106ae8c1dcaa4b993c8`).

## Packages

Choose `amd64` for Intel/AMD 64-bit PCs or `arm64` for ARM64 Linux desktops. Debian/Ubuntu users can install the matching `.deb` with `sudo apt install ./<package>.deb`. Other distributions should download the matching `.tar.gz` and `install-linux-<arch>.sh` to the same directory, then run `sh install-linux-<arch>.sh`. Alternatively, extract the archive and run `./clash-traffic`.

This is a desktop application and requires GTK 3 and WebKitGTK 4.1 (Debian/Ubuntu packages `libgtk-3-0` or `libgtk-3-0t64`, plus `libwebkit2gtk-4.1-0`). It is not a headless server or OpenWrt package. The Debian package declares these dependencies.

## EasyTier synchronization

1. Run a compatible mihomo/Clash controller and connect EasyTier with a virtual IPv4 interface.
2. Install a matching `easytier-cli` that supports `-o json route`; put it on PATH or specify its full path in Settings.
3. Enable EasyTier synchronization and use the same independent pairing key of at least 16 characters on each device. Do not use your EasyTier network password.
4. Detect EasyTier. Linux tries local RPC ports 15999 and 15888; specify the actual RPC endpoint and instance when necessary. Windows service/configuration discovery is Windows-only.
5. Allow TCP 47321 only between your own EasyTier peers, keep clocks synchronized, and exclude EasyTier from proxy interception.

The Linux and Windows feature builds use the same synchronization protocol. Statistics are synchronized; detailed access logs are not. Real multi-device interoperability must be verified in the target network.

Feature builds disable upstream automatic updates, so an upstream update cannot silently remove EasyTier support. Exit the original Clash Traffic before starting this build: both use the same application identity and data directory.

## Rebuild

Use the versions pinned by `go.mod` and `bun.lock` (the original build used Go 1.27.1 and Bun 1.4.4).

```sh
bun install --frozen-lockfile
CGO_ENABLED=0 go test ./...
bun run test:web
bun x mygo vet .
CLASH_TRAFFIC_EASYTIER_BUILD=1 CGO_ENABLED=0 bun run build --platform linux/amd64,linux/arm64
```

After building, run `python tools/package_linux.py` to normalize executable permissions (required for Windows cross-builds). Ship each architecture's `install.sh` alongside its archive, renamed to `install-linux-<arch>.sh`.

Packages are written under `build/`. Check release checksums before installation.