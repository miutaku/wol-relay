#!/bin/sh
set -eu

APP_NAME="wol-relay"
INSTALL_DIR="${HOME}/.local/bin"
CONFIG_DIR="${HOME}/.config/wol-relay"
DESKTOP_FILE="${HOME}/.local/share/applications/wol-relay.desktop"

REMOVE_CONFIG="${REMOVE_CONFIG:-0}"

rm -f "${INSTALL_DIR}/${APP_NAME}"
rm -f "$DESKTOP_FILE"

if [ "$REMOVE_CONFIG" = "1" ]; then
  rm -rf "$CONFIG_DIR"
fi

echo "Removed ${APP_NAME} from ${INSTALL_DIR}"
if [ "$REMOVE_CONFIG" = "1" ]; then
  echo "Removed config: ${CONFIG_DIR}"
else
  echo "Config kept: ${CONFIG_DIR}"
fi
