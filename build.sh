#!/usr/bin/env bash
set -euo pipefail

CURR_DIR="nginx_ui"
export CGO_ENABLED=1

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

mkdir -p "${CURR_DIR}/web" "${CURR_DIR}/swagger" "${CURR_DIR}/conf"

go build -ldflags="-w -s -n -v" -o "${OUTPUT}"
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
WorkingDirectory=/opt/${CURR_DIR}
ExecStart=/opt/${CURR_DIR}/$(basename "${OUTPUT}")
StandardOutput=journal+console
StandardError=journal+console
Restart=on-failure
RestartSec=3s

[Install]
WantedBy=multi-user.target
EOF

echo "Service file created: ${SERVICE_FILE}"

cp -f ./conf/app_prod.conf "./${CURR_DIR}/conf/app.conf"

if [[ -d ./web/build ]]; then
  mkdir -p "./${CURR_DIR}/web/build"
  cp -a ./web/build/. "./${CURR_DIR}/web/build/"
fi

if [[ -d ./swagger ]]; then
  cp -a ./swagger/. "./${CURR_DIR}/swagger/"
fi

tar -cvf base_cas.tar "./${CURR_DIR}"
