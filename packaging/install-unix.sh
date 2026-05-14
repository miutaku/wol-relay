#!/bin/sh
set -eu

APP_NAME="wol-relay"
INSTALL_DIR="${HOME}/.local/bin"
CONFIG_DIR="${HOME}/.config/wol-relay"
SERVICE_DIR="${HOME}/.config/systemd/user"
DESKTOP_FILE="${HOME}/.local/share/applications/wol-relay.desktop"
SERVICE_FILE="${SERVICE_DIR}/wol-relay.service"
INSTALL_MODE="${INSTALL_MODE:-gui}"

case "$INSTALL_MODE" in
  gui|agent) ;;
  *)
    echo "INSTALL_MODE must be gui or agent: ${INSTALL_MODE}" >&2
    exit 1
    ;;
esac

mkdir -p "$INSTALL_DIR" "$CONFIG_DIR"
cp "./${APP_NAME}" "${INSTALL_DIR}/${APP_NAME}"
chmod 0755 "${INSTALL_DIR}/${APP_NAME}"

if [ ! -f "${CONFIG_DIR}/wol-relay.json" ]; then
  "${INSTALL_DIR}/${APP_NAME}" init -config "${CONFIG_DIR}/wol-relay.json"
fi

if [ "$INSTALL_MODE" = "agent" ]; then
  rm -f "$DESKTOP_FILE"
  mkdir -p "$SERVICE_DIR"
  cat > "$SERVICE_FILE" <<EOF
[Unit]
Description=wol-relay lightweight Wake on LAN agent
After=network-online.target
Wants=network-online.target

[Service]
ExecStart=${INSTALL_DIR}/${APP_NAME} agent -config ${CONFIG_DIR}/wol-relay.json -light
Restart=on-failure
RestartSec=5s

[Install]
WantedBy=default.target
EOF

  systemctl --user daemon-reload
  systemctl --user enable --now wol-relay.service

  if command -v loginctl >/dev/null 2>&1; then
    loginctl enable-linger "$(id -un)" >/dev/null 2>&1 || true
  fi

  echo "Installed ${APP_NAME} lightweight agent to ${INSTALL_DIR}/${APP_NAME}"
  echo "Config: ${CONFIG_DIR}/wol-relay.json"
  echo "Service: systemctl --user status wol-relay.service"
  exit 0
fi

if command -v systemctl >/dev/null 2>&1; then
  systemctl --user disable --now wol-relay.service >/dev/null 2>&1 || true
fi
rm -f "$SERVICE_FILE"
if command -v systemctl >/dev/null 2>&1; then
  systemctl --user daemon-reload >/dev/null 2>&1 || true
fi

mkdir -p "${HOME}/.local/share/applications"
cat > "$DESKTOP_FILE" <<EOF
[Desktop Entry]
Type=Application
Name=wol-relay
Exec=${INSTALL_DIR}/${APP_NAME} gui -config ${CONFIG_DIR}/wol-relay.json
Terminal=false
Categories=Network;
EOF

echo "Installed ${APP_NAME} to ${INSTALL_DIR}/${APP_NAME}"
echo "Config: ${CONFIG_DIR}/wol-relay.json"
