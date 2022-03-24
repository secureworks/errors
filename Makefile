PACKAGES = $(shell go list ./...)

.DEFAULT_GOAL := help
.PHONY: help lint test

help:
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "\033[36m%-10s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

lint: ## Run go vet and golangci-lint.
	go vet ./... &&
	golangci-lint run ./...

test: ## Run tests.
	go test -short -v ./... -race

flake: ## Run test flake.
	go test -short -v ./... -race -test.failfast -test.count 10

coverage: $(patsubst %,%.coverage,$(PACKAGES))
	@rm -f .gocoverage/cover.txt
	gocovmerge .gocoverage/*.out > coverage.txt
	go tool cover -html=coverage.txt -o .gocoverage/index.html
	go tool cover -func=coverage.txt

%.coverage:
	@[ -d .gocoverage ] || mkdir .gocoverage
	go test -covermode=count -coverprofile=.gocoverage/$(subst /,-,$*).out $* -v
