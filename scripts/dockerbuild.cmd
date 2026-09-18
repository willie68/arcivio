@echo off
cd backend

echo building service
docker build -f ./build/package/Dockerfile ./ -t mcs/arcivio:V1
docker run --name arcivio -p 9543:8443 -p 9080:8080 mcs/arcivio:V1
cd ..