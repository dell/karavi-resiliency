clean:
	go clean -cache

unit-test:
	go test -v -coverprofile=c.out ./...

godog:
	go clean -cache
	go test -v -coverprofile=c.out -test.run TestGoDog ./...

gocover:
	go tool cover -html=c.out