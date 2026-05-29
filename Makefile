.PHONY: build install test fmt vet clean

build:
	go build -ldflags "-X github.com/ciaranRoche/openshell-delegate/cmd.version=dev" -o bin/openshell-delegate .

install:
	go install -ldflags "-X github.com/ciaranRoche/openshell-delegate/cmd.version=dev" .

test:
	go test ./...

fmt:
	go fmt ./...

vet:
	go vet ./...

clean:
	rm -rf bin/
