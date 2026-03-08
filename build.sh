#!/usr/bin/env bash
set -euo pipefail

CURR_DIR="nginx-ui"
export CGO_ENABLED=1
export GIN_MODE=release


npx update-browserslist-db@latest  && pnpm install && pnpm build 

MSYS2_MINGW64_BIN="/c/tools/msys64/mingw64/bin"

if [[ $# -lt 1 ]]; then
  echo "Usage: ./build.sh [win|linux]"
  exit 1
fi

target="$1"
case "$target" in
  win)
    export GOOS="windows"
    export GOARCH="amd64"
    # Keep MinGW runtime DLLs ahead of conflicting PATH entries for cgo/cc1.
    if [[ -x "${MSYS2_MINGW64_BIN}/gcc.exe" ]]; then
      export PATH="${MSYS2_MINGW64_BIN}:/c/Windows/System32:/c/Windows:${PATH}"
      export CC="${MSYS2_MINGW64_BIN}/gcc.exe"
      export CXX="${MSYS2_MINGW64_BIN}/g++.exe"
    fi
    OUTPUT="./${CURR_DIR}/${CURR_DIR}.exe"
    ;;
  linux)
    export GOOS="linux"
    export GOARCH="amd64"
    OUTPUT="./${CURR_DIR}/${CURR_DIR}_linux"
    ;;
  *)
    echo "Invalid argument. Use 'win' or 'linux'."
    exit 1
    ;;
esac
go generate
go build -tags=jsoniter -ldflags="-w -s -n -v" -o "${OUTPUT}"
echo "Build succeeded: ${OUTPUT}"

SERVICE_FILE="./${CURR_DIR}/${CURR_DIR}.service"
cat > "${SERVICE_FILE}" <<EOF
[Unit]
Description=The Base Cas Auth Server
After=network.target

[Service]
Type=simple
User=thebase
Group=thebase
Environment="GIN_MODE=release"
WorkingDirectory=/opt/${CURR_DIR}
ExecStart=/opt/${CURR_DIR}/$(basename "${OUTPUT}") -config /opt/${CURR_DIR}/app.ini
StandardOutput=journal+console
StandardError=journal+console
Restart=on-failure
RestartSec=3s

[Install]
WantedBy=multi-user.target
EOF

echo "Service file created: ${SERVICE_FILE}"

cp ./resources/base/app.example.ini "./${CURR_DIR}/app.ini"
cp ./resources/base/nginx_ui_sample.conf "./${CURR_DIR}/nginx_ui.conf"
cp ./resources/base/openapi.json "./${CURR_DIR}/openapi.json"

tar -cvf base_cas.tar "./${CURR_DIR}"
