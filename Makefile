GO_TEST_PACKAGES = $(shell cd robot && go list ./... | grep -v '/pkg/ai/zrt$$')

.PHONY: test-go test-go-full test-python test

test-go:
	cd robot && go test -timeout=2m $(GO_TEST_PACKAGES)

test-go-full:
	cd robot && go test ./...

test-python:
	python3 -m py_compile ai/balltracking/*.py ai/balltracking/localization/*.py

test: test-go test-python
