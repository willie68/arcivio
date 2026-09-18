cd backend
go test -coverprofile=cover.out -coverpkg=./... ./...
go tool cover -func cover.out
cd ..