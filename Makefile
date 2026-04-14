.PHONY: run build test lint health info clean

run:
	go run ./cmd/server

build:
	go build -o bin/server ./cmd/server

test:
	go test -json ./...

lint:
	golangci-lint run ./...

health:
	@curl -s http://localhost:8080/health | jq .

info:
	@echo '{"command":"info","status":"success","summary":"vk-investment-middleend","details":{"name":"vk-investment-middleend","type":"middleend","stack":"go","framework":"gin"}}'

clean:
	rm -rf bin/ tmp/
