@echo off
chcp 65001 >nul
echo ===== Real-time Voting System Development Environment =====

REM Environment variables
set DB_HOST=localhost
set DB_PORT=13306
set DB_USER=voteuser
set DB_PASSWORD=votepassword
set DB_NAME=votingdb
set REDIS_ADDR=localhost:16379
set REDIS_PASSWORD=redispassword
set REDIS_DB=0
set REDIS_MOCK=false

REM High concurrency config
set ENABLE_RATE_LIMIT=true
set GLOBAL_RATE_LIMIT=100
set USER_RATE_LIMIT=10

REM Server config
set SERVER_PORT=8090
set API_PREFIX=/api

REM Start middleware services (MySQL and Redis)
echo Starting middleware services...
docker-compose -f docker-compose-dev.yml up -d mysql redis

REM Wait for middleware to start
echo Waiting for middleware to start...
timeout /t 5 /nobreak >nul

REM Start backend server
echo Starting backend server...
cd backend
start cmd /k "go run main.go"
cd ..

REM Start frontend development server
echo Starting frontend development server...
cd frontend
start cmd /k "npm install && npm start"
cd ..

echo ===== All services started =====
echo Frontend dev server: http://localhost:3000
echo Backend API server: http://localhost:8090/api

echo Press any key to close this window...
pause >nul 