#!/usr/bin/make
# Makefile readme (ru): <http://linux.yaroslavl.ru/docs/prog/gnu_make_3-79_russian_manual.html>
# Makefile readme (en): <https://www.gnu.org/software/make/manual/html_node/index.html#SEC_Contents>

cwd = $(shell pwd)

SHELL = /bin/sh

DC_RUN_ARGS = --rm --user "$(shell id -u):$(shell id -g)"

.PHONY : help \
         fmt lint test cover psql shell
.DEFAULT_GOAL : help

help: ## Show this help
	@printf "\033[33m%s:\033[0m\n" 'Available commands'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  \033[32m%-11s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

fmt: ## Run source code formatter tools
	docker-compose run $(DC_RUN_ARGS) app sh -c 'GO111MODULE=off go get golang.org/x/tools/cmd/goimports && $$GOPATH/bin/goimports -d -w .'
	docker-compose run $(DC_RUN_ARGS) app gofmt -s -w -d .

lint: ## Run source code linters
	docker-compose run $(DC_RUN_ARGS) app go vet ./...
	docker-compose run --rm golint golangci-lint run

test: ## Run tests
	docker-compose run $(DC_RUN_ARGS) app go test -v -race -timeout 5s ./...

cover: ## Run tests with coverage report
	docker-compose run $(DC_RUN_ARGS) app sh -c 'go test -race -covermode=atomic -coverprofile /tmp/cp.out ./... && go tool cover -html=/tmp/cp.out -o ./coverage.html'
	-sensible-browser ./coverage.html && sleep 2 && rm -f ./coverage.html

shell: ## Start shell into container with golang
	docker-compose run $(DC_RUN_ARGS) app bash
