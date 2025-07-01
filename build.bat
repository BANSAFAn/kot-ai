@echo off
echo Building KOT.AI...

REM Check if Go is installed
where go >nul 2>nul
if %ERRORLEVEL% neq 0 (
    echo Error: Go is not installed. Please install Go from https://golang.org/dl/
    exit /b 1
)

REM Check Go version
for /f "tokens=3" %%g in ('go version') do set GOVERSION=%%g
echo Using %GOVERSION%

REM Download dependencies
echo Downloading dependencies...
go mod download
if %ERRORLEVEL% neq 0 (
    echo Error: Failed to download dependencies
    exit /b 1
)

REM Build the project
echo Building project...
go build -v -o kot.exe -ldflags="-H=windowsgui" . > build_output.txt 2>&1
dir >> build_output.txt
if %ERRORLEVEL% neq 0 (
    echo Error: Build failed
    exit /b 1
)

REM Check if the executable was created
if not exist kot.exe (
    echo Error: Executable file was not created
    exit /b 1
)

echo Build completed successfully!
echo Executable file: %CD%\kot.exe

pause