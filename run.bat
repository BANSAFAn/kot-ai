@echo off
echo Starting KOT.AI...

REM Check if executable exists
if not exist kot.exe (
    echo Executable file not found. Starting build process...
    call build.bat
    if %ERRORLEVEL% neq 0 (
        echo Error: Build process failed
        pause
        exit /b 1
    )
)

REM Check if executable is valid
if not exist kot.exe (
    echo Error: Executable file still not found after build
    pause
    exit /b 1
)

REM Start the application
start /b kot.exe
if %ERRORLEVEL% neq 0 (
    echo Error: Failed to start KOT.AI
    pause
    exit /b 1
)

echo KOT.AI started successfully! Web interface available at http://localhost:8080/