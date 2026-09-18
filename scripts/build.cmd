@echo off
setlocal
cd /d "%~dp0.."

echo building frontend
cd frontend
call npm run build
if errorlevel 1 exit /b 1
cd ..

echo building backend
cd backend
swag init -d "./internal/adapter/inbound/http/apiv1" -g "../../../../../cmd/service/main.go" --parseDependency --parseDepth 2 -o "./api"
if errorlevel 1 exit /b 1
go build -ldflags="-s -w" -o bin/arcivio.exe cmd/service/main.go
if errorlevel 1 exit /b 1
cd ..