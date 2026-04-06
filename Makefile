PLUGIN_NAME = out_zerobus
GO_SRC = $(wildcard *.go)

.PHONY: build test lint clean docker

build: $(PLUGIN_NAME).so

$(PLUGIN_NAME).so: $(GO_SRC) go.mod go.sum
	CGO_ENABLED=1 go build -buildmode=c-shared -o $@ .

test:
	go test -v -count=1 ./...

lint:
	golangci-lint run ./...

clean:
	rm -f $(PLUGIN_NAME).so $(PLUGIN_NAME).h

docker:
	docker build -t fluent-bit-zerobus .
