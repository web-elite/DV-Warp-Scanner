# Improvements

- Upgraded the bundled Xray-core to the official v26.11.9 release.
- Added a numbered management menu for scanning and maintenance.
- Added an Xray-core manager to install the latest stable release or an exact version tag.
- Added installed-version switching, including a fallback to the bundled Xray-core.
- Added SHA-256 verification for downloaded Xray archives before activation.
- Limited scan concurrency to 64 workers and split large scans into batches of 256 endpoints.
- Reused a single WARP registration across scan batches and added an Xray readiness check.
- Improved Xray process cleanup and Cloudflare API response validation.
- Fixed the Xray loglevel configuration field and reset scan settings between runs.
- Separated the Scanner version from the Xray-core version.
- Added automated Go tests and static analysis before release builds.
- Updated cross-platform Xray archive selection for Linux, Windows and macOS.
- Renamed the distributed executable and installer to DV-Warp-Scanner.
- Updated the Linux/Termux installer and documentation to use this repository's releases.
