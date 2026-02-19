#!/bin/sh

set -e

BINARY="filehound"
REPO="ripkitten-co/filehound"
VERSION="${VERSION:-latest}"

get_latest_version() {
  curl -sI "https://github.com/${REPO}/releases/latest" | \
    grep -i "location:" | \
    sed 's/.*tag\/\(.*\).*/\1/' | \
    tr -d '\r\n'
}

if [ "$VERSION" = "latest" ]; then
  VERSION=$(get_latest_version)
fi

OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case "$ARCH" in
  x86_64|amd64) ARCH="amd64" ;;
  aarch64|arm64) ARCH="arm64" ;;
  *)
    echo "Unsupported architecture: $ARCH"
    exit 1
    ;;
esac

URL="https://github.com/${REPO}/releases/download/${VERSION}/${BINARY}_${VERSION#v}_${OS}_${ARCH}.tar.gz"

echo "Downloading ${BINARY} ${VERSION} for ${OS}/${ARCH}..."

if ! curl -fsSL "$URL" | tar xz -C /tmp; then
  echo "Failed to download. Binary may not exist for this platform."
  exit 1
fi

if [ -w /usr/local/bin ]; then
  mv /tmp/${BINARY} /usr/local/bin/${BINARY}
  chmod +x /usr/local/bin/${BINARY}
  echo "Installed to /usr/local/bin/${BINARY}"
else
  echo "Installing to ~/.local/bin..."
  mkdir -p ~/.local/bin
  mv /tmp/${BINARY} ~/.local/bin/${BINARY}
  chmod +x ~/.local/bin/${BINARY}
  echo "Add ~/.local/bin to your PATH"
fi

echo "Done! Run '${BINARY} --help' to get started."
