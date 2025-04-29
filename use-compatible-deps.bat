@echo off
chcp 65001 >nul
echo ===== Using Compatible Dependencies =====

cd backend

echo Updating gin-contrib/cors to a compatible version...
go get github.com/gin-contrib/cors@v1.5.0

echo Updating gin framework to a compatible version...
go get github.com/gin-gonic/gin@v1.9.1

echo Tidying go.mod...
go mod tidy

echo ===== Dependencies updated =====
echo Now you can try running rebuild-and-start.bat again.

cd ..
pause 