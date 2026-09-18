set CC=clang
cd backend
go test --race ./...
cd ..