build:
	go build ./...

run: build
	go run cmd/computeshader/main.go
