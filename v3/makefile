run:
	go run ./cmd/main.go

test:
	go test ./... -v -cover -coverprofile=coverage.out

view-coverage:
	go tool cover -func=coverage.out

html-coverage:
	go tool cover -html=coverage.out -o coverage.html

.PHONY: test run

