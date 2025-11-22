.PHONY: help server client build clean deps test

help:
	@echo "Available commands:"
	@echo "  make deps      - Download dependencies"
	@echo "  make server    - Run the server"
	@echo "  make client    - Run the client"
	@echo "  make build     - Build server and client binaries"
	@echo "  make clean     - Remove built binaries"
	@echo "  make test      - Run tests"

deps:
	go mod download
	go mod tidy

server:
	go run cmd/server/main.go

client:
	go run cmd/client/main.go

client-termloop:
	go run cmd/client/main.go -mode game -room default-room -name Player1 -termloop

build:
	@mkdir -p bin
	go build -o bin/server cmd/server/main.go
	go build -o bin/client cmd/client/main.go
	@echo "Binaries built in bin/"

clean:
	rm -rf bin/

test:
	go test ./...

# Development helpers
dev-server:
	go run cmd/server/main.go -addr :8080

dev-client1:
	go run cmd/client/main.go -mode game -room dev-room -name Alice

dev-client2:
	go run cmd/client/main.go -mode game -room dev-room -name Bob
