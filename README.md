<div align="center">

<img src="resources/icon.png" width="112" alt="">

# Clash Traffic

> EasyTier feature fork: [Linux packages, setup and reproducible build](docs/LINUX.md). Upstream automatic updates are disabled in feature builds.

See where your Clash Verge traffic goes — by app, site, proxy node, subscription and device.

**English** · [简体中文](README.zh-CN.md)

<picture><source media="(prefers-color-scheme: dark)" srcset="docs/screenshots/overview-dark.png"><img alt="Overview: seven days of traffic across three computers" src="docs/screenshots/overview-light.png"></picture>

</div>

## Features

- **Every app and site** — upload, download and connections per app and host. Helper processes count toward their app; subdomains can merge into their domain.
- **Proxy or direct** — one switch shows only the traffic your proxy plan counts.
- **Subscriptions** — what each subscription carried, beside the usage and expiry its provider reports.
- **All your computers, no setup** — copies running in your Tailscale tailnet find each other and merge their numbers. A shared folder or WebDAV works too.
- **EasyTier discovery** — find direct and relayed devices automatically and reuse the same incremental statistics sync, with a separate pairing key. See [EasyTier setup](#easytier-setup).
- **Access log and live view** — every connection with its app, target, node and rule; live speed and open connections.
- **Stays out of the way** — finds Clash Verge by itself, lives in the menu bar, starts at login, updates itself, exports CSV, follows light and dark mode.

Your data stays on your computers, which exchange it directly. The app speaks English, 简体中文, 繁體中文, 日本語, 한국어 and Русский, following your system or the language you pick in Settings.

## Screenshots

<table>
<tr><td width="50%" valign="top"><picture><source media="(prefers-color-scheme: dark)" srcset="docs/screenshots/apps-dark.png"><img alt="Apps" src="docs/screenshots/apps-light.png"></picture><br><sub><b>Apps</b> — and the sites one of them visits</sub></td><td width="50%" valign="top"><picture><source media="(prefers-color-scheme: dark)" srcset="docs/screenshots/subscriptions-dark.png"><img alt="Subscriptions" src="docs/screenshots/subscriptions-light.png"></picture><br><sub><b>Subscriptions</b> — provider usage beside your own count</sub></td></tr>
<tr><td width="50%" valign="top"><picture><source media="(prefers-color-scheme: dark)" srcset="docs/screenshots/live-dark.png"><img alt="Live connections" src="docs/screenshots/live-light.png"></picture><br><sub><b>Live</b> — speed and open connections</sub></td><td width="50%" valign="top"><picture><source media="(prefers-color-scheme: dark)" srcset="docs/screenshots/records-dark.png"><img alt="Access log" src="docs/screenshots/records-light.png"></picture><br><sub><b>Access log</b> — every connection, searchable</sub></td></tr>
<tr><td width="50%" valign="top"><picture><source media="(prefers-color-scheme: dark)" srcset="docs/screenshots/devices-dark.png"><img alt="Devices" src="docs/screenshots/devices-light.png"></picture><br><sub><b>Devices</b> — found over Tailscale</sub></td><td width="50%"></td></tr>
</table>

## Install

Download the latest build from [Releases](https://github.com/lufficc/clash-traffic/releases): a universal `.dmg` for macOS 13 or later (Apple silicon and Intel), an installer for Windows (x64 or ARM64), or a `.deb` / `.tar.gz` for Linux.

The builds are not signed yet. On macOS, right-click the app and choose **Open** the first time; on Windows, choose **More info → Run anyway**.

It works with Clash Verge Rev, or any mihomo core whose controller it can reach. To tell apps apart it sets the core's `find-process-mode` to `always`; add `find-process-mode: always` to Clash Verge's merge config to keep it after a reload.

## EasyTier setup

1. Run EasyTier with a virtual IPv4 interface, encrypted networking, and a local management RPC endpoint (GUI service normally uses `127.0.0.1:15999`; standalone core normally uses `127.0.0.1:15888`). Install a matching `easytier-cli` supporting `-o json route`.
2. In Settings → Multi-device sync, enable EasyTier and enter the same dedicated sync key (at least 16 characters) on your devices. Do not use the EasyTier network password. Selecting EasyTier switches off Tailscale sharing; existing Tailscale settings and stored statistics are retained.
3. The CLI is found on PATH, beside the app, in the running EasyTier installation, or in common installation directories. Keep an existing working CLI path. Leave RPC and instance overrides empty: Windows discovery reads running service/process metadata, service config-directory instance IDs/names, and (as a fallback) local listener ports. It queries instances individually, so older multi-instance services also work. Only a single usable network is selected automatically; if several are found, use **Detect EasyTier → Use this network**, then Save. A working explicit endpoint remains preferred; failed endpoints fall back to discovery.
4. Open Devices → EasyTier sync. Discovery and incremental sync run every minute; Scan now refreshes immediately. Allow inbound TCP **47321** only on the EasyTier interface/from your devices (development builds use 47322). Exclude the EasyTier network and its transport from proxy capture.

Only virtual IPv4 addresses in the selected instance's route table are probed, including relayed routes. Subnet proxy CIDRs and public relay transport addresses are not scanned. The service binds only to the selected virtual IP and requires both a discovered source IP and the matching key. It does not follow HTTP redirects. Statistics are shared; detailed access logs are not. CLI failure stops serving at the next scan. History remains available while devices are offline.

The detection button reports the CLI version, detected RPC/instance, and sanitized CLI errors (including stderr instead of just exit status 1). Probes are read-only and never start/stop services or change GUI settings. Empty/default endpoint fallbacks are 15999 then 15888. Windows service introspection is Windows-only; other platforms retain CLI/default-port probing.

The implementation uses the official EasyTier CLI route JSON (`ipv4`, `hostname`, `next_hop_hostname`, `path_len`). Both flat arrays (2.5/2.6 style) and per-instance result envelopes (newer CLI) are supported. Versions without this format and no-TUN/IPv6-only configurations are unsupported. Additional UI labels are translated into Simplified Chinese; other non-English locales currently show the new labels in English.

Requests and responses use HMAC-SHA256; the sync key is not sent to discovered peers. Requests are bound to their destination and expire after 60 seconds, so keep device clocks synchronized. The key is stored in the app's local settings file, alongside existing credentials; keep it private when sharing backups.

For a local Windows feature build with upstream automatic updates disabled:

```powershell
$env:CLASH_TRAFFIC_EASYTIER_BUILD = "1"
bun run build
```

The feature build retains the original application identifier and data location; exit the original app before launching it. Do not run both against the same data directory.

## Build

With Go 1.27+ and Bun:

```sh
bun install
bun run dev        # run with live reload
bun run build      # package for this machine into build/
go test ./...      # and bun run test:web
```

Built with [MyGo](https://mygo.egoist.dev/).
