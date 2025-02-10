
.PHONY: all dep lint vet test test-coverage build clean

all: build

run: ## Run Server
	@cd optimization && make run

