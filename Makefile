.PHONY: build zip publish

test:
	go test ./...

coverage:
	go test -coverprofile=c.out ./...
	go tool cover -html=c.out -o coverage.html
	open coverage.html

cleanup:
	rm c.out
	rm coverage.html
