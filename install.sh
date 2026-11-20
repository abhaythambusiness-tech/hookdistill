#!/bin/sh
# Installs the latest hookdistill release into ~/.local/bin (no sudo).
# Verifies the SHA-256 checksum before installing anything.
#   VERSION=0.1.0       install a specific version (without the leading v)
#   INSTALL_DIR=...     change the target directory
set -eu

REPO="abhaythambusiness-tech/hookdistill"
BASE_URL="${HOOKDISTILL_BASE_URL:-https://github.com/$REPO/releases/download}"
INSTALL_DIR="${INSTALL_DIR:-$HOME/.local/bin}"

fail() { echo "install.sh: $*" >&2; exit 1; }

case "$(uname -s)" in
  Linux) os=linux ;;
  Darwin) os=darwin ;;
  *) fail "unsupported OS $(uname -s); on Windows download the .zip from https://github.com/$REPO/releases" ;;
esac
case "$(uname -m)" in
  x86_64 | amd64) arch=amd64 ;;
  aarch64 | arm64) arch=arm64 ;;
  *) fail "unsupported CPU $(uname -m)" ;;
esac

if [ -z "${VERSION:-}" ]; then
  tag=$(curl -fsSL -o /dev/null -w '%{url_effective}' "https://github.com/$REPO/releases/latest") || fail "could not look up the latest release"
  VERSION="${tag##*/v}"
fi
[ -n "$VERSION" ] || fail "could not determine the version"

archive="hookdistill_${VERSION}_${os}_${arch}.tar.gz"
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

curl -fsSL -o "$tmp/$archive" "$BASE_URL/v$VERSION/$archive" || fail "download failed: $archive"
curl -fsSL -o "$tmp/checksums.txt" "$BASE_URL/v$VERSION/checksums.txt" || fail "download failed: checksums.txt"

want=$(awk -v f="$archive" '$2 == f { print $1 }' "$tmp/checksums.txt")
[ -n "$want" ] || fail "no checksum listed for $archive"
if command -v sha256sum >/dev/null 2>&1; then
  got=$(sha256sum "$tmp/$archive" | awk '{ print $1 }')
elif command -v shasum >/dev/null 2>&1; then
  got=$(shasum -a 256 "$tmp/$archive" | awk '{ print $1 }')
else
  fail "need sha256sum or shasum to verify the download"
fi
[ "$want" = "$got" ] || fail "checksum mismatch for $archive: refusing to install"

tar -xzf "$tmp/$archive" -C "$tmp" hookdistill
mkdir -p "$INSTALL_DIR"
install -m 755 "$tmp/hookdistill" "$INSTALL_DIR/hookdistill"
echo "installed hookdistill $VERSION to $INSTALL_DIR/hookdistill"
case ":$PATH:" in
  *":$INSTALL_DIR:"*) ;;
  *) echo "note: $INSTALL_DIR is not on your PATH" ;;
esac
