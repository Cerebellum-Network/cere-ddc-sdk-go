packages = ./core/... ./contract/...

.PHONY: test
test:
	go test ${packages}

lint:
	docker run --rm -v ${PWD}:/app -w /app golangci/golangci-lint:v1.50 golangci-lint run ${packages}
