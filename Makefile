build:
	@go build -o bin/gobank

run: build
	@./bin/gobank

dev:
	@nodemon --watch './**/*.go' --signal SIGTERM --exec go run main.go