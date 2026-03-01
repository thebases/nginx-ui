@ECHO OFF
REM Get the current folder name
SET CURR_DIR=nginx_ui
SET CGO_ENABLED=1
SET "MSYS2_MINGW64_BIN=C:\tools\msys64\mingw64\bin"

REM Keep MinGW runtime DLLs ahead of conflicting PATH entries for cgo/cc1.
IF EXIST "%MSYS2_MINGW64_BIN%\gcc.exe" (
    SET "PATH=%MSYS2_MINGW64_BIN%;C:\Windows\System32;C:\Windows;%PATH%"
    SET "CC=%MSYS2_MINGW64_BIN%\gcc.exe"
    SET "CXX=%MSYS2_MINGW64_BIN%\g++.exe"
)


REM Check if an argument is provided
IF "%~1"=="" (
    ECHO "Usage: build.bat [win|linux]"
    EXIT /B 1
)

REM Set target based on argument
IF "%~1"=="win" (
    set GOOS=windows
    set GOARCH=amd64
    set OUTPUT=.\%CURR_DIR%\%CURR_DIR%.exe
) ELSE IF "%~1"=="linux" (
    set GOOS=linux
    set GOARCH=amd64
    set OUTPUT=./%CURR_DIR%/%CURR_DIR%_linux
) ELSE (
    ECHO Invalid argument. Use 'win' or 'linux'.
    EXIT /B 1
)

REM Create the build directory if it doesn't exist
IF NOT EXIST %CURR_DIR% (
    mkdir %CURR_DIR%
)
IF NOT EXIST %CURR_DIR%\web (
    mkdir %CURR_DIR%\web
)
IF NOT EXIST %CURR_DIR%\swagger (
    mkdir %CURR_DIR%\swagger
)
IF NOT EXIST %CURR_DIR%\conf (
    mkdir %CURR_DIR%\conf
)

REM Build the Go project
go build -ldflags="-w -s -n -v" -o %OUTPUT%
IF ERRORLEVEL 1 (
    ECHO Build failed.
    EXIT /B 1
) ELSE (
    ECHO Build succeeded: %OUTPUT%
)


REM Generate the service file
SET SERVICE_FILE=.\%CURR_DIR%\%CURR_DIR%.service
(
    ECHO [Unit]
    ECHO Description=The Base Cas Auth Server
    ECHO After=network.target
    ECHO.
    ECHO [Service]
    ECHO Type=simple
    ECHO User=thebase
    ECHO Group=thebase
    ECHO WorkingDirectory=/opt/%CURR_DIR%
    ECHO ExecStart=/opt/%CURR_DIR%/%OUTPUT%
    ECHO StandardOutput=journal+console
    ECHO StandardError=journal+console
    ECHO Restart=on-failure
    ECHO RestartSec=3s
    ECHO.
    ECHO [Install]
    ECHO WantedBy=multi-user.target
) > %SERVICE_FILE%

ECHO Service file created: %SERVICE_FILE%
COPY .\conf\app_prod.conf .\%CURR_DIR%\conf\app.conf
xcopy ".\web\build\*" "%CURR_DIR%\web\build\" /E /I /H /C /Y
xcopy ".\swagger\*" "%CURR_DIR%\swagger\" /E /I /H /C /Y
tar -cvf base_cas.tar .\%CURR_DIR%
