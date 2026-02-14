.PHONY: build run

build:
	go build -buildvcs=false -o ./tmp/main .

run: build
	./tmp/main
	