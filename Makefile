.PHONY: test system coverage fmt lint build clean

test:
	go test -race -count=1 ./...

system:
	@echo "system tests not yet implemented"

coverage:
	./scripts/check_coverage.sh

fmt:
	gofmt -s -w .

lint:
	go vet ./...

build:
	go build -o tinyclaw ./cmd/tinyclaw/

clean:
	rm -f tinyclaw tinyclawd
	rm -rf bundle-*
