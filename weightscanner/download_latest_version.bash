#!/bin/bash
set -e

REPO="andrea-aus-hh/renpho-es-cs20m"
DEST_DIR="."
TARBALL="weightscanner.tar.gz"

ASSET_URL=$(curl -s https://api.github.com/repos/$REPO/releases/latest \
  | grep "browser_download_url.*$TARBALL" \
  | cut -d '"' -f 4)

if [ -z "$ASSET_URL" ]; then
  echo "Error: Latest release tarball not found!"
  exit 1
fi

mkdir -p "$DEST_DIR"

echo "Downloading latest release from $ASSET_URL..."
curl -L -o "$DEST_DIR/$TARBALL" "$ASSET_URL"

echo "Extracting $DEST_DIR/$TARBALL..."
tar -xzvf "$DEST_DIR/$TARBALL" -C "$DEST_DIR"
