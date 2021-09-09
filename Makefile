test:
	go test -race ./...

test.cover:
	go test -race -coverprofile=coverage.out ./...
	#go tool cover -func=coverage.out
	#go tool cover -html=coverage.out