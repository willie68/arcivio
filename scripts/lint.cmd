@echo off
cd backend

cls
revive -config revive.toml -formatter friendly ./...
cd ..