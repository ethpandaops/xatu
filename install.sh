#!/bin/bash

set -e

error() {
  echo "$1" >&2
  exit 1
}

info() {
  echo "$1"
}

version_ge() {
  [ "$(printf '%s\n' "$1" "$2" | sort -V | tail -n1)" = "$1" ]
}

OS="$(uname | tr '[:upper:]' '[:lower:]')"
case "$OS" in
  darwin*) OS="darwin" ;;
  linux*) OS="linux" ;;
  *) error "Unsupported OS: $OS" ;;
esac
info "Detected OS: $OS"

ARCH="$(uname -m)"
case "$ARCH" in
  x86_64 | amd64) ARCH="amd64" ;;
  arm64 | aarch64) ARCH="arm64" ;;
  *) error "Unsupported architecture: $ARCH" ;;
esac
info "Detected Architecture: $ARCH"

REPO_OWNER="ethpandaops"
REPO_NAME="xatu"
BASE_INSTALL_PATH="${BASE_INSTALL_PATH:-/usr/local/bin}"
STANDARD_BINARY_NAME="${REPO_NAME}"
INSTALL_PATH="${BASE_INSTALL_PATH}/${STANDARD_BINARY_NAME}"

fetch_latest_tag() {
  info "Fetching the latest release information from GitHub..."
  LATEST_RELEASE_JSON=$(curl -s "https://api.github.com/repos/${REPO_OWNER}/${REPO_NAME}/releases/latest") || error "Failed to fetch release information."

  TAG=$(echo "$LATEST_RELEASE_JSON" | grep -oP '"tag_name":\s*"\K(.*)(?=")') || error "Failed to parse release tag."
  [[ -z "$TAG" ]] && error "Release tag not found."
  info "Latest release tag: $TAG"
}

get_installed_version() {
  if command -v ${INSTALL_PATH} >/dev/null 2>&1; then
    VERSION_OUTPUT=$(${INSTALL_PATH} version 2>/dev/null) || {
      info "xatu is installed but failed to retrieve version information."
      return 1
    }

    if [[ $VERSION_OUTPUT =~ Xatu/v([0-9]+\.[0-9]+\.[0-9]+)-([a-f0-9]+) ]]; then
      INSTALLED_VERSION="${BASH_REMATCH[1]}"
      info "Installed xatu version: $INSTALLED_VERSION"
      return 0
    else
      info "Failed to parse xatu version from output: $VERSION_OUTPUT"
      return 1
    fi
  else
    info "xatu is not installed or not found in PATH."
    return 1
  fi
}

check_existing_installation() {
  if which xatu >/dev/null 2>&1; then
    if [ "$(which xatu)" != "$INSTALL_PATH" ]; then
      info "xatu is already installed at $(which xatu) but expecting it at $INSTALL_PATH. You might want to remove it."
    fi
  fi
}

check_version() {
  if get_installed_version; then
    LATEST_VERSION="${TAG#v}"
    info "Latest released xatu version: $LATEST_VERSION"

    if version_ge "$INSTALLED_VERSION" "$LATEST_VERSION"; then
      info "xatu is already up to date (version $INSTALLED_VERSION, location: ${INSTALL_PATH}). Skipping installation."
      exit 0
    else
      info "A newer version of xatu is available (installed: $INSTALLED_VERSION, latest: $LATEST_VERSION). Proceeding with installation."
    fi
  else
    info "xatu is not installed or version could not be determined. Proceeding with installation."
  fi
}

uninstall() {
  if [ -f "$INSTALL_PATH" ]; then
    info "Uninstalling xatu from $INSTALL_PATH..."
    sudo rm "$INSTALL_PATH" || error "Failed to remove xatu from $INSTALL_PATH"
    info "xatu has been successfully uninstalled."
  else
    info "xatu is not installed at $INSTALL_PATH. Nothing to uninstall."
  fi
  exit 0
}

if [ "$1" = "uninstall" ]; then
  uninstall
fi

check_existing_installation
fetch_latest_tag
check_version

ASSET_NAME="${REPO_NAME}_${TAG#v}_${OS}_${ARCH}.tar.gz"
info "Constructed asset name: $ASSET_NAME"

DOWNLOAD_URL="https://github.com/${REPO_OWNER}/${REPO_NAME}/releases/download/${TAG}/${ASSET_NAME}"
info "Download URL: $DOWNLOAD_URL"

TMP_DIR=$(mktemp -d) || error "Failed to create temporary directory."
info "Created temporary directory: $TMP_DIR"

cleanup() {
  rm -rf "$TMP_DIR"
}
trap cleanup EXIT

info "Downloading ${ASSET_NAME}..."
curl -L -o "${TMP_DIR}/${ASSET_NAME}" "$DOWNLOAD_URL" || error "Failed to download the asset."

info "Extracting the binary..."
tar -xzf "${TMP_DIR}/${ASSET_NAME}" -C "$TMP_DIR" || error "Failed to extract the archive."

BINARY_PATTERN="${REPO_NAME}-*"
BINARY_FILE=$(find "$TMP_DIR" -type f -name "$BINARY_PATTERN" | head -n 1)

[[ -z "$BINARY_FILE" ]] && error "No binary matching pattern '${BINARY_PATTERN}' found in the archive."

info "Found binary: $(basename "$BINARY_FILE")"

RENAMED_BINARY="${TMP_DIR}/${STANDARD_BINARY_NAME}"
info "Renaming binary to: $STANDARD_BINARY_NAME"
mv "$BINARY_FILE" "$RENAMED_BINARY" || error "Failed to rename the binary."

info "Installing the binary to ${INSTALL_PATH}..."
sudo mv "$RENAMED_BINARY" "$INSTALL_PATH" || error "Failed to move the binary to ${INSTALL_PATH}."

sudo chmod +x "$INSTALL_PATH" || error "Failed to make the binary executable."

info "Installation successful! '${REPO_NAME}' has been installed to ${INSTALL_PATH}."
info "You can run it using '${REPO_NAME}' from your terminal."

exit 0
