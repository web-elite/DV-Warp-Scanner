# 💦 DV Warp Scanner

A cross-platform Cloudflare WARP endpoint scanner powered by the official [Xray-core](https://github.com/XTLS/Xray-core). It measures real connection latency through WARP and helps you find endpoints that perform well on your network.

This project is based on [BPB Warp Scanner](https://github.com/bia-pain-bache/BPB-Warp-Scanner). The original project and its contributors are acknowledged and credited.

## Features

- Tests network quality separately for IPv4 and IPv6 to tune scan settings.
- Registers one WARP configuration per scan and reuses it across batches.
- Processes large scans in small Xray batches with a bounded worker pool to keep memory and process usage under control.
- Measures real connection latency instead of relying on ICMP ping.
- Exports results to CSV.
- Supports IPv4, IPv6, and dual-stack scanning.
- Offers Quick, Normal, Deep, and Custom scan sizes.
- Supports optional UDP noise with configurable packet type, size, delay, and count.
- Provides a numbered management menu for scanning and maintenance.
- Checks for the latest stable Xray-core release and installs exact release tags on demand.
- Keeps multiple Xray-core versions available and supports switching back to the bundled version.
- Verifies Xray-core downloads with SHA-256 before activation or packaging.

## Download

Download the archive for your operating system and architecture from the [latest release](https://github.com/web-elite/DV-Warp-Scanner/releases/latest), extract it, and run `DV-Warp-Scanner`.

## Linux and Android (Termux)

> **Android users:** Install Termux from its [official GitHub releases](https://github.com/termux/termux-app/releases). The Google Play version may be outdated.

Run this command from the directory where you want to install the Scanner:

```bash
bash <(curl -fsSL https://raw.githubusercontent.com/web-elite/DV-Warp-Scanner/main/install.sh)
```

The installer detects your CPU architecture, downloads the matching release, and updates an existing installation when a newer version is available.

## Before scanning

- Disconnect your VPN before starting a scan.
- On Windows, fully exit v2rayN or another VPN client from the system tray. Clearing the system proxy alone may not be enough.
- For the most representative UDP-noise results, use noise settings similar to those configured in your WARP client or panel.

## Build from source

Requirements: Go 1.24 or newer, Make, curl, unzip, and (for packaging) zip.

At runtime, open the management menu and choose **Xray-core Manager** to check for the latest stable release, install an exact release tag, or switch between installed versions. The currently bundled version remains available as a fallback.

The Xray-core version used in a build is pinned in `XRAY_VERSION`. To build for the current platform:

```bash
make build VERSION=$(cat VERSION)
```

To build for a specific target:

```bash
make build VERSION=$(cat VERSION) GOOS=linux GOARCH=amd64
```

## Credits

- Original project: [bia-pain-bache/BPB-Warp-Scanner](https://github.com/bia-pain-bache/BPB-Warp-Scanner)
- Core engine: [XTLS/Xray-core](https://github.com/XTLS/Xray-core)

This project is distributed under the GNU General Public License v3.0. See [LICENSE](LICENSE).
