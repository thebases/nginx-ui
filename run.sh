#!/usr/bin/env bash

set -euo pipefail

export CGO_ENABLED=1
MSYS2_MINGW64_BIN="${MSYS2_MINGW64_BIN:-/c/tools/msys64/mingw64/bin}"

# Keep MinGW runtime DLLs ahead of conflicting PATH entries for cgo/cc1.
if [[ -x "${MSYS2_MINGW64_BIN}/gcc.exe" || -x "${MSYS2_MINGW64_BIN}/gcc" ]]; then
  export PATH="${MSYS2_MINGW64_BIN}:/c/Windows/System32:/c/Windows:${PATH}"
  if [[ -x "${MSYS2_MINGW64_BIN}/gcc.exe" ]]; then
    export CC="${MSYS2_MINGW64_BIN}/gcc.exe"
  else
    export CC="${MSYS2_MINGW64_BIN}/gcc"
  fi
  if [[ -x "${MSYS2_MINGW64_BIN}/g++.exe" ]]; then
    export CXX="${MSYS2_MINGW64_BIN}/g++.exe"
  else
    export CXX="${MSYS2_MINGW64_BIN}/g++"
  fi
fi

export GIN_MODE=debug
go run main.go -config ./app.ini %1
