.PHONY: build run dev swag lint test clean

build: swag
	go build -buildvcs=false -o ./tmp/main .

run: build
	./tmp/main

dev: swag
	air

swag:
	swag init

lint:
	go vet ./...

test:
	go test ./...

clean:
	rm -rf tmp
