#!/usr/bin/env bash
set -euo pipefail

REPOSITORY="web-elite/DV-Warp-Scanner"
BINARY="DV-Warp-Scanner"

OS=$(uname -s)
if [ "$OS" != "Linux" ]; then
    echo "This installer supports Linux and Android (Termux) only."
    exit 1
fi

ARCH=$(uname -m)
case "$ARCH" in
    aarch64|arm64) ARCH="arm64" ;;
    armv7*|armv8*) ARCH="arm" ;;
    x86_64)        ARCH="amd64" ;;
    i386|i686)     ARCH="386" ;;
    *)             echo "Unsupported architecture: ${ARCH}" && exit 1 ;;
esac

ARCHIVE="${BINARY}-linux-${ARCH}.tar.gz"
LATEST_VERSION=$(curl -fsSL "https://raw.githubusercontent.com/${REPOSITORY}/main/VERSION")

if [ -x "./${BINARY}" ]; then
    INSTALLED_VERSION=$("./${BINARY}" --version)
    echo "Installed version: ${INSTALLED_VERSION}"
    echo "Latest version: ${LATEST_VERSION}"

    if [ "${INSTALLED_VERSION}" = "${LATEST_VERSION}" ]; then
        echo "Scanner is up to date. Running..."
        exec "./${BINARY}"
    fi

    echo "Updating to version ${LATEST_VERSION}..."
else
    echo "Scanner not found. Installing version ${LATEST_VERSION}..."
fi

TMP_ARCHIVE=$(mktemp)
trap 'rm -f "$TMP_ARCHIVE"' EXIT

echo "Downloading ${ARCHIVE}..."
curl -fL --retry 3 --retry-delay 2 \
    -o "$TMP_ARCHIVE" \
    "https://github.com/${REPOSITORY}/releases/download/${LATEST_VERSION}/${ARCHIVE}"

tar -xzf "$TMP_ARCHIVE"
chmod +x "./${BINARY}"

echo "Installation completed. Starting scanner..."
exec "./${BINARY}"
