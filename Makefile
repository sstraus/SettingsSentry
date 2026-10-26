.PHONY: build test lint clean release help dmg zip coverage integration-test act-test act-lint

BINARY_NAME=settingssentry
VERSION=$(shell git describe --tags --always --dirty)
LDFLAGS=-ldflags "-X main.Version=${VERSION}"
GORELEASER_ARGS?=release --snapshot --clean

help:
	@echo "Available commands:"
	@echo "  make build            - Build the application"
	@echo "  make test             - Run tests"
	@echo "  make integration-test - Run integration tests"
	@echo "  make coverage         - Generate test coverage report"
	@echo "  make lint             - Run linter"
	@echo "  make act-test         - Run tests with act (GitHub Actions locally)"
	@echo "  make act-lint         - Run linter with act (GitHub Actions locally)"
	@echo "  make clean            - Remove build artifacts"
	@echo "  make release          - Create a new release"
	@echo "  make dmg              - Create a macOS DMG installer"
	@echo "  make zip              - Create a zip archive"
	@echo "  make help             - Show this help message"

build:
	@set -e; \
	if [ -f configs/TestCommand.cfg ]; then \
		test ! -e configs/TestCommand.not_embed; \
		mv configs/TestCommand.cfg configs/TestCommand.not_embed; \
		trap 'mv configs/TestCommand.not_embed configs/TestCommand.cfg' EXIT; \
	fi; \
	go build ${LDFLAGS} -o ${BINARY_NAME} .

test:
	go test -p 1 -v -race ./...

integration-test:
	go test -p 1 -v -tags=integration ./...

coverage:
	go test -p 1 -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out

lint:
	@if ! command -v golangci-lint &> /dev/null; then \
		echo "golangci-lint not found, installing..."; \
		brew install golangci-lint; \
	fi
	golangci-lint run ./...

clean:
	go clean
	rm -f ${BINARY_NAME}
	rm -rf dist/
	rm -f *.dmg
	rm -f *.zip
	rm -f coverage.out

release:
	@set -e; \
	if [ -f configs/TestCommand.cfg ]; then \
		test ! -e configs/TestCommand.not_embed; \
		mv configs/TestCommand.cfg configs/TestCommand.not_embed; \
		trap 'mv configs/TestCommand.not_embed configs/TestCommand.cfg' EXIT; \
	fi; \
	goreleaser ${GORELEASER_ARGS}

dmg:
	mkdir -p ./${BINARY_NAME}.app/Contents/MacOS
	cp ${BINARY_NAME} ./${BINARY_NAME}.app/Contents/MacOS/
	hdiutil create -volname "${BINARY_NAME}" -srcfolder ./${BINARY_NAME}.app -ov -format UDZO ${BINARY_NAME}.dmg

zip: build
	rm -f ${BINARY_NAME}.zip
	zip ${BINARY_NAME}.zip ${BINARY_NAME} configs/*.cfg -x configs/TestCommand.cfg

install: build
	cp ${BINARY_NAME} /usr/local/bin/

uninstall:
	rm -f /usr/local/bin/${BINARY_NAME}

act-test:
	@if ! command -v act &> /dev/null; then \
		echo "act not found. Install it with: brew install act"; \
		exit 1; \
	fi
	act push -j test

act-lint:
	@if ! command -v act &> /dev/null; then \
		echo "act not found. Install it with: brew install act"; \
		exit 1; \
	fi
	act push -j lint
