#!/usr/bin/env bash
# Downloads the pinned mcp-file-tools release binary on first run, verifies its
# SHA-256, caches it, and execs it. Later runs reuse the cache.
# Launched by Claude Code as: bash run.sh (it spawns servers without a shell,
# so a bare ./run.sh won't work).
set -euo pipefail

REPO="dimitar-grigorov/mcp-file-tools"
VERSION="v1.7.0"   # bump on each plugin release

# CLAUDE_PLUGIN_DATA survives plugin updates; fall back to a local cache if unset.
DATA_DIR="${CLAUDE_PLUGIN_DATA:-$HOME/.cache/mcp-file-tools}/bin"
mkdir -p "$DATA_DIR"

os="$(uname -s | tr '[:upper:]' '[:lower:]')"
case "$os" in
  msys*|mingw*|cygwin*|win*) os="windows" ;;   # Git Bash reports MINGW64_NT-...
  darwin)                    os="darwin"  ;;
  linux)                     os="linux"   ;;
  *) echo "mcp-file-tools: unsupported OS '$os'" >&2; exit 1 ;;
esac

arch="$(uname -m)"
case "$arch" in
  x86_64|amd64)  arch="amd64" ;;
  arm64|aarch64) arch="arm64" ;;
  *) echo "mcp-file-tools: unsupported arch '$arch'" >&2; exit 1 ;;
esac

ext=""
[ "$os" = "windows" ] && ext=".exe"

asset="mcp-file-tools_${os}_${arch}${ext}"
bin="${DATA_DIR}/mcp-file-tools-${VERSION}-${os}-${arch}${ext}"

http_download() {  # $1=dest $2=url
  if command -v curl >/dev/null 2>&1; then curl -fsSL -o "$1" "$2"
  elif command -v wget >/dev/null 2>&1; then wget -qO "$1" "$2"
  else echo "mcp-file-tools: need curl or wget" >&2; return 1; fi
}

sha256() {  # print hex digest of $1
  if command -v sha256sum >/dev/null 2>&1; then sha256sum "$1" | cut -d' ' -f1
  elif command -v shasum >/dev/null 2>&1; then shasum -a 256 "$1" | cut -d' ' -f1
  elif command -v openssl >/dev/null 2>&1; then openssl dgst -sha256 "$1" | sed 's/.*= *//'
  else echo "mcp-file-tools: no sha256 tool found" >&2; return 1; fi
}

if [ ! -x "$bin" ]; then
  base="https://github.com/${REPO}/releases/download/${VERSION}"
  tmp="$(mktemp -d)"
  trap 'rm -rf "$tmp"' EXIT

  echo "mcp-file-tools: downloading ${VERSION} (${os}/${arch})..." >&2
  http_download "$tmp/$asset" "$base/$asset"
  http_download "$tmp/checksums.txt" "$base/checksums.txt"

  want="$(grep " ${asset}\$" "$tmp/checksums.txt" | cut -d' ' -f1)"
  got="$(sha256 "$tmp/$asset")"
  if [ -z "$want" ] || [ "$want" != "$got" ]; then
    echo "mcp-file-tools: checksum mismatch for $asset (want=$want got=$got)" >&2
    exit 1
  fi

  chmod +x "$tmp/$asset"
  mv "$tmp/$asset" "$bin"
fi

# Directories come from the client via the MCP roots protocol, so no args needed.
exec "$bin" "$@"
