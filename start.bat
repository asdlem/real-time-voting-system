@echo off
chcp 65001 >nul
echo ===== Real-time Voting System Startup =====

REM Check if docker is installed
where docker >nul 2>nul
if %ERRORLEVEL% NEQ 0 (
    echo Docker is not installed, please install Docker first.
    exit /b
)

REM Start services
echo Starting Docker containers...
docker-compose up -d

REM Wait for services to start
echo Waiting for services to start...
timeout /t 5 /nobreak >nul

REM Check service status
echo Checking service status:
docker-compose ps

echo ===== Service startup complete =====
echo Frontend URL: http://localhost
echo Backend API URL: http://localhost:8090/api

pause 