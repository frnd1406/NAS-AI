.PHONY: build run stop clean test

build:
	go build -o bin/orchestrator .

run: build
	./bin/orchestrator

stop:
	pkill -f bin/orchestrator || true

clean:
	rm -rf bin/

test:
	go test -v ./...

dev: build
	API_URL=http://localhost:8080 ./bin/orchestrator

prod: build
	API_URL=https://your-domain.com ./bin/orchestrator
