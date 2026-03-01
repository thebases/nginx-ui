set CGO_ENABLED=1
set "MSYS2_MINGW64_BIN=C:\tools\msys64\mingw64\bin"

REM Keep MinGW runtime DLLs ahead of conflicting PATH entries for cgo/cc1.
if exist "%MSYS2_MINGW64_BIN%\gcc.exe" (
    set "PATH=%MSYS2_MINGW64_BIN%;C:\Windows\System32;C:\Windows;%PATH%"
    set "CC=%MSYS2_MINGW64_BIN%\gcc.exe"
    set "CXX=%MSYS2_MINGW64_BIN%\g++.exe"
)
set GIN_MODE=debug
go run main.go
