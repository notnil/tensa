GO_TEST_PACKAGES = $(shell cd robot && go list ./... | grep -v '/pkg/ai/zrt$$')

.PHONY: test-go test-go-native test-go-hardware test-python test

test-go:
	cd robot && go test -timeout=2m $(GO_TEST_PACKAGES)

test-go-native:
	cd robot && go test ./...

test-go-hardware:
	cd robot && go test -tags=hardware ./...

test-python:
	python3 -m py_compile ai/balltracking/*.py ai/balltracking/localization/*.py

test: test-go test-python
