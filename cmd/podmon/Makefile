# Includes the following generated file to get semantic version information
MAJOR=0
MINOR=0
PATCH=54
VERSION?="v$(MAJOR).$(MINOR).$(PATCH)"
REGISTRY?="${REGISTRY_HOST}:${REGISTRY_PORT}/podmon"

clean:
	go clean ./...

build:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64  go build -a -ldflags '-w' ./...

docker:
	docker build --no-cache -t "$(REGISTRY):$(VERSION)" --label commit=$(shell git log --max-count 1 --format="%H") .

push:
	docker push "$(REGISTRY):$(VERSION)"

unit-test:
	go test -v -coverprofile=c.out ./...

godog:
	go clean -cache
	go test -v -coverprofile=c.out -test.run TestMain ./...

