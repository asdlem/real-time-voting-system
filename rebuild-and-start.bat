@echo off
chcp 65001 >nul
echo ===== Real-time Voting System Startup =====

REM Check if docker is installed
where docker >nul 2>nul
if %ERRORLEVEL% NEQ 0 (
    echo Docker is not installed, please install Docker first.
    exit /b 1
)

REM Stop and remove existing containers
echo Stopping any running containers...
docker-compose down
if %ERRORLEVEL% NEQ 0 (
    echo Warning: Failed to stop containers, but continuing...
)

REM Remove existing images and volumes
echo Removing existing containers...
docker-compose rm -f
if %ERRORLEVEL% NEQ 0 (
    echo Warning: Failed to remove containers, but continuing...
)

REM Clean existing images
echo Cleaning existing images...
docker image prune -f
if %ERRORLEVEL% NEQ 0 (
    echo Warning: Failed to clean images, but continuing...
)

REM Try build up to 3 times
set BUILD_ATTEMPTS=0
:BUILD_RETRY
set /a BUILD_ATTEMPTS+=1
echo Build attempt %BUILD_ATTEMPTS% of 3...

REM Build new images
echo Building new images...
docker-compose build --no-cache
if %ERRORLEVEL% NEQ 0 (
    if %BUILD_ATTEMPTS% LSS 3 (
        echo Retrying build...
        goto BUILD_RETRY
    ) else (
        echo Error: Failed to build images after 3 attempts. Please check the error messages above.
        pause
        exit /b 1
    )
)

REM Start services
echo Starting Docker containers...
docker-compose up -d
if %ERRORLEVEL% NEQ 0 (
    echo Error: Failed to start containers. Please check the error messages above.
    pause
    exit /b 1
)

REM Wait for services to start
echo Waiting for services to start (15 seconds)...
timeout /t 15 /nobreak >nul

REM Check service status
echo Checking service status:
docker-compose ps

echo ===== Service startup complete =====
echo Frontend URL: http://localhost
echo Backend API URL: http://localhost:8090/api

pause 