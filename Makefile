.PHONY: test fmt vet lint lint-new preflight install-hooks

test:
	go test ./... -count=1

fmt:
	go fmt ./...

vet:
	go vet ./...

lint:
	golangci-lint run ./...

lint-new:
	golangci-lint run --new-from-rev=HEAD ./...

check-imports:
	@echo "Checking forbidden imports..."
	@! grep -r '"github.com/dpopsuev/origami' --include='*.go' . && \
	 ! grep -r '"github.com/dpopsuev/djinn' --include='*.go' . && \
	 echo "OK: no forbidden imports"

preflight: fmt vet lint check-imports test

install-hooks:
	@echo '#!/bin/sh' > .git/hooks/pre-commit
	@echo 'make lint-new' >> .git/hooks/pre-commit
	@chmod +x .git/hooks/pre-commit
	@echo "pre-commit hook installed (runs make lint-new)"
