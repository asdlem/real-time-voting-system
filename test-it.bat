@echo off
chcp 65001 >nul
echo ===== Real-time Voting System Test Script =====

REM Check if docker is installed
where docker >nul 2>nul
if %ERRORLEVEL% NEQ 0 (
    echo Docker is not installed, please install Docker first.
    exit /b 1
)

REM Start only MySQL and Redis services
echo Starting MySQL and Redis...
docker-compose -f docker-compose-dev.yml up -d mysql redis
if %ERRORLEVEL% NEQ 0 (
    echo Error: Failed to start MySQL and Redis.
    pause
    exit /b 1
)

REM Wait for services to start
echo Waiting for services to start (10 seconds)...
timeout /t 10 /nobreak >nul

REM Set environment variables
echo Setting environment variables...
set DB_HOST=localhost
set DB_PORT=13306
set DB_USER=voteuser
set DB_PASSWORD=votepassword
set DB_NAME=votingdb
set REDIS_ADDR=localhost:16379
set REDIS_PASSWORD=redispassword
set REDIS_DB=0
set REDIS_MOCK=false
set ROCKETMQ_MOCK=true
set SERVER_PORT=8090
set API_PREFIX=/api
set GIN_MODE=debug

REM Start backend server
echo Starting backend server...
cd backend
start cmd /k "go run main.go"
cd ..

REM Wait for backend to start
echo Waiting for backend to start (10 seconds)...
timeout /t 10 /nobreak >nul

REM Run test script
echo Running test script...
python tests/Multi_user_testing.py --poll-id 1 --mode 4 --users 5 --dup-users 3 --dup-attempts 2 --reset

echo ===== Test complete =====
echo You can now test the application manually at:
echo Backend API: http://localhost:8090/api

pause 