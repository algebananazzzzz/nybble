#!/bin/sh
# Install the latest `nybble` release binary on macOS.
#   curl -fsSL https://raw.githubusercontent.com/algebananazzzzz/nybble/main/install.sh | sh
set -eu

OWNER="algebananazzzzz"
REPO="nybble"
BIN="nybble"

err() { echo "install: $*" >&2; exit 1; }

# --- platform detection ---------------------------------------------------
os=$(uname -s)
[ "$os" = "Darwin" ] || err "only macOS is supported (got $os)"

case "$(uname -m)" in
  arm64)  arch="arm64" ;;
  x86_64) arch="amd64" ;;
  *)      err "unsupported architecture: $(uname -m)" ;;
esac

for tool in curl tar shasum; do
  command -v "$tool" >/dev/null 2>&1 || err "missing required tool: $tool"
done

# --- resolve latest release ----------------------------------------------
api="https://api.github.com/repos/${OWNER}/${REPO}/releases/latest"
tag=$(curl -fsSL "$api" | grep '"tag_name"' | head -1 | sed -E 's/.*"tag_name":[[:space:]]*"([^"]+)".*/\1/')
[ -n "$tag" ] || err "could not determine latest release tag from $api"
version=${tag#v}

asset="${BIN}_${version}_darwin_${arch}.tar.gz"
base="https://github.com/${OWNER}/${REPO}/releases/download/${tag}"
echo "install: downloading ${asset} (${tag})"

tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

curl -fsSL "${base}/${asset}"        -o "${tmp}/${asset}" || err "download failed: ${base}/${asset}"
curl -fsSL "${base}/checksums.txt"   -o "${tmp}/checksums.txt" || err "checksums download failed"

# --- verify checksum ------------------------------------------------------
want=$(grep " ${asset}\$" "${tmp}/checksums.txt" | awk '{print $1}')
[ -n "$want" ] || err "no checksum for ${asset} in checksums.txt"
got=$(shasum -a 256 "${tmp}/${asset}" | awk '{print $1}')
[ "$want" = "$got" ] || err "checksum mismatch (want $want, got $got)"

# --- extract + install ----------------------------------------------------
tar -xzf "${tmp}/${asset}" -C "$tmp"
[ -f "${tmp}/${BIN}" ] || err "archive did not contain ${BIN}"

if [ -w /usr/local/bin ] 2>/dev/null; then
  dest="/usr/local/bin"
elif [ "$(id -u)" = "0" ]; then
  dest="/usr/local/bin"
else
  dest="${HOME}/.local/bin"
  mkdir -p "$dest"
fi

install -m 0755 "${tmp}/${BIN}" "${dest}/${BIN}"
echo "install: installed ${BIN} ${tag} -> ${dest}/${BIN}"

case ":${PATH}:" in
  *":${dest}:"*) ;;
  *) echo "install: add ${dest} to your PATH" ;;
esac

cat <<'EOF'

Next steps:
  1. Install the login helper (REQUIRED for `nybble auth`):
       npm i -g @playwright/cli && playwright-cli install chromium
  2. Set the deployment-specific endpoints (see README "Configuration"),
     e.g. in ~/.config/nybble/.env:
       NYBBLE_API_BASE="https://<host>/<app-path>"
       NYBBLE_LOGIN_URL="https://<host>/<login-path>"
  3. Log in once:   nybble auth
  4. Run the TUI:   nybble
EOF
