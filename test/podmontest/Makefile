# Includes the following generated file to get semantic version information
MAJOR=0
MINOR=0
PATCH=54
VERSION?="v$(MAJOR).$(MINOR).$(PATCH)"
REGISTRY?="${REGISTRY_HOST}:${REGISTRY_PORT}/podmontest"

clean:
	go clean ./...

build:
	rm -f podmontest
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -tags podmontest -ldflags '-w' -o podmontest *.go

docker:
	docker build -t "$(REGISTRY):$(VERSION)" --label commit=$(shell git log --max-count 1 --format="%H") .

push:
	docker push "$(REGISTRY):$(VERSION)"


