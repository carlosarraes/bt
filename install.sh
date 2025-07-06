#!/bin/sh
# bt installer
#
# Usage:
#   curl -sSf https://raw.githubusercontent.com/carlosarraes/bt/main/install.sh | sh

set -e

REPO="carlosarraes/bt"
BINARY_NAME="bt"
BIN_DIR="${BIN_DIR:-$HOME/.local/bin}"
GITHUB_LATEST="https://api.github.com/repos/${REPO}/releases/latest"

get_arch() {
  # detect architecture
  ARCH=$(uname -m)
  case $ARCH in
  x86_64) ARCH="amd64" ;;
  aarch64) ARCH="arm64" ;;
  arm64) ARCH="arm64" ;;
  *)
    echo "Unsupported architecture: $ARCH"
    exit 1
    ;;
  esac
}

get_os() {
  # detect os
  OS=$(uname -s)
  case $OS in
  Linux) OS="linux" ;;
  Darwin) OS="darwin" ;;
  *)
    echo "Unsupported OS: $OS"
    exit 1
    ;;
  esac
}

download_binary() {
  # get latest release info
  echo "Fetching latest release..."
  VERSION=$(curl -s $GITHUB_LATEST | grep -o '"tag_name": "[^"]*' | cut -d'"' -f4)
  if [ -z "$VERSION" ]; then
    echo "Failed to fetch latest version"
    exit 1
  fi

  echo "Latest version: $VERSION"

  # create temporary directory
  TMP_DIR=$(mktemp -d)
  # Ensure cleanup happens even if script fails or exits early
  trap 'rm -rf "$TMP_DIR"' EXIT

  echo "Downloading ${BINARY_NAME} ${VERSION}..."

  # Construct archive name based on OS and architecture
  ARCHIVE_NAME="${BINARY_NAME}-${OS}-${ARCH}.tar.gz"
  
  DOWNLOAD_URL="https://github.com/${REPO}/releases/download/${VERSION}/${ARCHIVE_NAME}"
  echo "Downloading from: $DOWNLOAD_URL"
  curl -fsSL "$DOWNLOAD_URL" -o "${TMP_DIR}/${ARCHIVE_NAME}" || {
    echo "Download failed. Check URL/permissions/network."
    exit 1
  }

  # extract archive
  cd "$TMP_DIR"
  tar -xzf "$ARCHIVE_NAME" || {
    echo "Failed to extract archive"
    exit 1
  }

  # find the binary (should be bt-OS-ARCH after extraction)
  BINARY_FILE="${BINARY_NAME}-${OS}-${ARCH}"
  if [ ! -f "$BINARY_FILE" ]; then
    echo "Binary file $BINARY_FILE not found in archive"
    exit 1
  fi

  # make it executable
  chmod +x "$BINARY_FILE"

  # Check if BIN_DIR exists and create if needed
  CREATED_DIR_MSG=""
  if [ ! -d "$BIN_DIR" ]; then
    echo "Installation directory '$BIN_DIR' not found."
    echo "Creating directory: $BIN_DIR"
    mkdir -p "$BIN_DIR"
    CREATED_DIR_MSG="Note: Created directory '$BIN_DIR'. You might need to add it to your system's PATH."
  fi

  # install binary (no sudo needed for $HOME/.local/bin)
  echo "Installing to $BIN_DIR..."
  install -m 755 "$BINARY_FILE" "$BIN_DIR/$BINARY_NAME"

  # cleanup happens via trap

  echo "${BINARY_NAME} ${VERSION} installed successfully to $BIN_DIR"

  # Print the warning message if the directory was created
  if [ -n "$CREATED_DIR_MSG" ]; then
    echo ""
    echo "$CREATED_DIR_MSG"
  fi
}

# Run the installer
get_arch
get_os
download_binary

echo ""
echo "Installation complete! Run '${BINARY_NAME} --help' to get started."
echo "Example usage: ${BINARY_NAME} pr list"