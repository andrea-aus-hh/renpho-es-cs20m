#!/bin/bash
set -e

REPO="andrea-aus-hh/renpho-es-cs20m"
DEST_DIR="/home/andrea/weightscanner/bin"
TARBALL="weightscanner.tar.gz"

if [ -f "$DEST_DIR/$TARBALL" ]; then
    echo "Removing existing tarball: $DEST_DIR/$TARBALL"
    rm "$DEST_DIR/$TARBALL"
fi

if [ -d "$DEST_DIR" ]; then
    echo "Removing existing extracted files in: $DEST_DIR"
    rm -rf "$DEST_DIR/*"
fi

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

exec "$DEST_DIR/weightscanner"
