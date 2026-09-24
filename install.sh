#!/bin/sh

set -eu

PROJECT_URL="https://gitlab.com/1001n/kapellmeister"
BINARY_NAME="kapellmeister"
ASSET_NAME="kapellmeister-linux-amd64"
CHECKSUM_NAME="${ASSET_NAME}.sha256"
INSTALL_DIR="${KAPELLMEISTER_INSTALL_DIR:-/usr/local/bin}"
VERSION="${KAPELLMEISTER_VERSION:-}"

say() {
    printf '%s\n' "$*"
}

fail() {
    printf 'kapellmeister installer: %s\n' "$*" >&2
    exit 1
}

usage() {
    cat <<'EOF'
Install Kapellmeister on Linux x86_64.

Usage:
  install.sh [--version VERSION] [--install-dir DIRECTORY]

Options:
  --version VERSION       Install a release tag such as v1.2.3.
  --install-dir DIRECTORY Install into DIRECTORY (default: /usr/local/bin).
  -h, --help              Show this help.

Environment variables:
  KAPELLMEISTER_VERSION
  KAPELLMEISTER_INSTALL_DIR
EOF
}

while [ "$#" -gt 0 ]; do
    case "$1" in
        --version)
            [ "$#" -ge 2 ] || fail "--version requires a value"
            VERSION=$2
            shift 2
            ;;
        --install-dir)
            [ "$#" -ge 2 ] || fail "--install-dir requires a value"
            INSTALL_DIR=$2
            shift 2
            ;;
        -h|--help)
            usage
            exit 0
            ;;
        *)
            fail "unknown option: $1"
            ;;
    esac
done

[ -n "$INSTALL_DIR" ] || fail "install directory cannot be empty"
case "$INSTALL_DIR" in
    /*) ;;
    *) fail "install directory must be an absolute path" ;;
esac

if [ -n "$VERSION" ]; then
    case "$VERSION" in
        *[!A-Za-z0-9._-]*) fail "invalid version: $VERSION" ;;
    esac
fi

case "$(uname -s 2>/dev/null || true)" in
    Linux) ;;
    *) fail "only Linux is currently supported" ;;
esac

case "$(uname -m 2>/dev/null || true)" in
    x86_64|amd64) ;;
    *) fail "only the x86_64 architecture is currently supported" ;;
esac

if command -v curl >/dev/null 2>&1; then
    download() {
        curl -fsSL --retry 3 --connect-timeout 10 --output "$2" "$1"
    }
elif command -v wget >/dev/null 2>&1; then
    download() {
        wget -qO "$2" "$1"
    }
else
    fail "curl or wget is required"
fi

command -v install >/dev/null 2>&1 || fail "the install utility is required"

TMP_DIR=$(mktemp -d 2>/dev/null) || fail "could not create a temporary directory"
trap 'rm -rf "$TMP_DIR"' EXIT HUP INT TERM

BINARY_FILE="$TMP_DIR/$ASSET_NAME"
CHECKSUM_FILE="$TMP_DIR/$CHECKSUM_NAME"

download_assets() {
    base_url=$1
    query=${2:-}
    download "$base_url/$ASSET_NAME$query" "$BINARY_FILE" &&
        download "$base_url/$CHECKSUM_NAME$query" "$CHECKSUM_FILE"
}

if [ -n "$VERSION" ]; then
    RELEASE_URL="$PROJECT_URL/-/releases/$VERSION/downloads"
    say "Downloading Kapellmeister $VERSION..."
    download_assets "$RELEASE_URL" || fail "release $VERSION was not found"
else
    RELEASE_URL="$PROJECT_URL/-/releases/permalink/latest/downloads"
    say "Downloading the latest Kapellmeister release..."
    if ! download_assets "$RELEASE_URL"; then
        say "No published release was found; using the latest successful main build."
        ARTIFACT_URL="$PROJECT_URL/-/jobs/artifacts/main/raw/dist"
        download_assets "$ARTIFACT_URL" "?job=build_binaries" ||
            fail "could not download Kapellmeister from GitLab"
    fi
fi

EXPECTED_HASH=$(awk 'NR == 1 { print $1 }' "$CHECKSUM_FILE")
[ "${#EXPECTED_HASH}" -eq 64 ] || fail "invalid SHA-256 checksum"
case "$EXPECTED_HASH" in
    *[!0-9A-Fa-f]*) fail "invalid SHA-256 checksum" ;;
esac

if command -v sha256sum >/dev/null 2>&1; then
    ACTUAL_HASH=$(sha256sum "$BINARY_FILE" | awk '{ print $1 }')
elif command -v shasum >/dev/null 2>&1; then
    ACTUAL_HASH=$(shasum -a 256 "$BINARY_FILE" | awk '{ print $1 }')
else
    fail "sha256sum or shasum is required to verify the download"
fi

[ "$EXPECTED_HASH" = "$ACTUAL_HASH" ] || fail "SHA-256 checksum verification failed"
chmod 0755 "$BINARY_FILE"

TARGET="$INSTALL_DIR/$BINARY_NAME"
if [ "$(id -u)" -eq 0 ]; then
    mkdir -p "$INSTALL_DIR"
    install -m 0755 "$BINARY_FILE" "$TARGET"
elif [ -d "$INSTALL_DIR" ] && [ -w "$INSTALL_DIR" ]; then
    install -m 0755 "$BINARY_FILE" "$TARGET"
elif command -v sudo >/dev/null 2>&1; then
    sudo mkdir -p "$INSTALL_DIR"
    sudo install -m 0755 "$BINARY_FILE" "$TARGET"
else
    fail "cannot write to $INSTALL_DIR; run as root or choose a writable --install-dir"
fi

say "Kapellmeister was installed to $TARGET"
say "Run: kapellmeister version"
