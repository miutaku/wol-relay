#!/usr/bin/env bash
set -euo pipefail

version="${VERSION:-0.0.0}"
plist_version="${version%%[-+]*}"
arch="${GOARCH:-$(go env GOARCH)}"
app_name="wol-relay"
display_name="wol-relay Wake on LAN"
bundle_id="io.github.miutaku.wol-relay"

dist_dir="dist/darwin-${arch}"
app_dir="${dist_dir}/${display_name}.app"
contents_dir="${app_dir}/Contents"
macos_dir="${contents_dir}/MacOS"
resources_dir="${contents_dir}/Resources"
dmg_root="${dist_dir}/dmg-root"
dmg_path="dist/wol-relay_${version}_darwin_${arch}.dmg"

rm -rf "${dist_dir}"
mkdir -p "${macos_dir}" "${resources_dir}"

CGO_ENABLED=1 GOOS=darwin GOARCH="${arch}" go build -tags nativegui -o "${macos_dir}/${app_name}" ./cmd/wol-relay
chmod +x "${macos_dir}/${app_name}"

cat > "${contents_dir}/Info.plist" <<PLIST
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>CFBundleDevelopmentRegion</key>
  <string>ja</string>
  <key>CFBundleDisplayName</key>
  <string>${display_name}</string>
  <key>CFBundleExecutable</key>
  <string>${app_name}</string>
  <key>CFBundleIdentifier</key>
  <string>${bundle_id}</string>
  <key>CFBundleInfoDictionaryVersion</key>
  <string>6.0</string>
  <key>CFBundleName</key>
  <string>${app_name}</string>
  <key>CFBundlePackageType</key>
  <string>APPL</string>
  <key>CFBundleShortVersionString</key>
  <string>${plist_version}</string>
  <key>CFBundleVersion</key>
  <string>${plist_version}</string>
  <key>LSMinimumSystemVersion</key>
  <string>12.0</string>
  <key>NSHighResolutionCapable</key>
  <true/>
</dict>
</plist>
PLIST

cp README.md LICENSE "${resources_dir}/"

mkdir -p "${dmg_root}"
cp -R "${app_dir}" "${dmg_root}/"
ln -s /Applications "${dmg_root}/Applications"

hdiutil create \
  -volname "${display_name}" \
  -srcfolder "${dmg_root}" \
  -ov \
  -format UDZO \
  "${dmg_path}"

echo "${dmg_path}"
