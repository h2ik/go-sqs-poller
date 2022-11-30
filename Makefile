.PHONY: all
all: up

.env:
		cp .env-dist .env

.PHONY: dep
dep:
		go mod tidy
		go mod vendor

.PHONY: up
up: dep .env

.PHONY: staticanalysis
staticanalysis:
		which staticcheck > /dev/null || go install honnef.co/go/tools/cmd/staticcheck@latest
		staticcheck ./...

.PHONY: unittests
unittests:
		go clean -testcache
		go test -mod vendor -timeout 300s -race -v ./...

.PHONY: test
test: unittests staticanalysis

.PHONY: test-report
test-report:
		CGO_ENABLED=1 go install gotest.tools/gotestsum@latest
		CGO_ENABLED=1 gotestsum --junitfile report.xml --format testname -- -mod vendor -timeout 300s -coverprofile=cover.out -race ./...

.PHONY: cover
cover: test-report
		go tool cover -func=cover.out

.PHONY: cover
cover-html: test-report
		go tool cover -html=cover.out
