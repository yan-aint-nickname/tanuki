typo:
	@typos

format:
	@gofmt -s -w $(CURDIR)

test:
	@go test -coverprofile=.coverage
	@go tool cover -html=.coverage -o coverage.html

lint:
	@golangci-lint run -v
