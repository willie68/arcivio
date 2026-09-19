@echo off
setlocal
cd /d "%~dp0.."

echo building frontend
cd frontend
call npm run build
if errorlevel 1 exit /b 1
cd ..

cd backend

echo building service
docker build -f ./build/package/Dockerfile ./ -t mcs/arcivio:V1
docker run --name arcivio -p 9543:8443 -p 9080:8080 -v ./data:/data mcs/arcivio:V1 
cd ..