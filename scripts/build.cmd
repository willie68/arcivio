@echo off
cd backend
swag init -d "./internal/adapter/inbound/http/apiv1" -g "../../../../../cmd/service/main.go" --parseDependency --parseDepth 2 -o "./api"
go build -ldflags="-s -w" -o bin/arcivio.exe cmd/service/main.go
cd ..