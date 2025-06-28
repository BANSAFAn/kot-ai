@echo off
echo Checking KOT.AI status...

REM Check if executable exists
if not exist kot.exe (
    echo Status: [ERROR] KOT.AI executable not found
    set error_found=true
) else (
    echo Status: [OK] KOT.AI executable found
    
    REM Try to use the built-in status check if executable exists
    echo.
    echo Running built-in status check...
    echo.
    kot.exe -status
    if %ERRORLEVEL% neq 0 (
        echo.
        echo Built-in status check failed, continuing with batch file checks...
        echo.
    ) else (
        exit /b 0
    )
)

REM Check if application is running
tasklist /FI "IMAGENAME eq kot.exe" 2>NUL | find /I /N "kot.exe">NUL
if "%ERRORLEVEL%"=="0" (
    echo Status: [OK] KOT.AI is currently running
) else (
    echo Status: [INFO] KOT.AI is not currently running
)

REM Check configuration directory
if exist "%USERPROFILE%\.kot.ai" (
    echo Status: [OK] Configuration directory exists
    
    REM Check configuration file
    if exist "%USERPROFILE%\.kot.ai\config.json" (
        echo Status: [OK] Configuration file exists
    ) else (
        echo Status: [WARNING] Configuration file not found
    )
    
    REM Check history database
    if exist "%USERPROFILE%\.kot.ai\history.db" (
        echo Status: [OK] History database exists
    ) else (
        echo Status: [INFO] History database not found
    )
) else (
    echo Status: [WARNING] Configuration directory not found
)

REM Check Go installation
where go >nul 2>nul
if %ERRORLEVEL% neq 0 (
    echo Status: [WARNING] Go is not installed
) else (
    for /f "tokens=3" %%g in ('go version') do set GOVERSION=%%g
    echo Status: [OK] Go is installed (Version: %GOVERSION%)
)

echo.

REM Provide summary based on findings
if defined error_found (
    echo Status: CRITICAL - Application may not function correctly
    echo.
    echo Recommendations:
    echo - Run build.bat to create the executable
    exit /b 1
) else (
    echo Status: HEALTHY - All systems operational
    echo.
    echo Status check completed.
    exit /b 0
)

pause