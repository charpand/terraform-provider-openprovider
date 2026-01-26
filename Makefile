.PHONY: all bootstrap lint test format docs

all: format lint test docs

bootstrap:
	./scripts/bootstrap

lint:
	./scripts/lint

test:
	./scripts/test

format:
	./scripts/format

docs:
	./scripts/docs
