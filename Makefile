.PHONY: test-go test-python test

test-go:
	cd robot && go test ./...

test-python:
	python3 -m py_compile ai/balltracking/*.py ai/balltracking/localization/*.py

test: test-go test-python
