#!/bin/sh
set -eu

APP_NAME="wol-relay"
INSTALL_DIR="${HOME}/.local/bin"
CONFIG_DIR="${HOME}/.config/wol-relay"

mkdir -p "$INSTALL_DIR" "$CONFIG_DIR"
mkdir -p "${HOME}/.local/share/applications"
cp "./${APP_NAME}" "${INSTALL_DIR}/${APP_NAME}"
chmod 0755 "${INSTALL_DIR}/${APP_NAME}"

if [ ! -f "${CONFIG_DIR}/wol-relay.json" ]; then
  "${INSTALL_DIR}/${APP_NAME}" init -config "${CONFIG_DIR}/wol-relay.json"
fi

cat > "${HOME}/.local/share/applications/wol-relay.desktop" <<EOF
[Desktop Entry]
Type=Application
Name=wol-relay
Exec=${INSTALL_DIR}/${APP_NAME} gui -config ${CONFIG_DIR}/wol-relay.json
Terminal=false
Categories=Network;
EOF

echo "Installed ${APP_NAME} to ${INSTALL_DIR}/${APP_NAME}"
echo "Config: ${CONFIG_DIR}/wol-relay.json"
