.PHONY: build test lint clean release help dmg zip coverage integration-test

BINARY_NAME=settingssentry
VERSION=$(shell git describe --tags --always --dirty)
LDFLAGS=-ldflags "-X main.Version=${VERSION}"

help:
	@echo "Available commands:"
	@echo "  make build            - Build the application"
	@echo "  make test             - Run tests"
	@echo "  make integration-test - Run integration tests"
	@echo "  make coverage         - Generate test coverage report"
	@echo "  make lint             - Run linter"
	@echo "  make clean            - Remove build artifacts"
	@echo "  make release          - Create a new release"
	@echo "  make dmg              - Create a macOS DMG installer"
	@echo "  make zip              - Create a zip archive"
	@echo "  make help             - Show this help message"

build:
	@echo "Temporarily renaming TestCommand.cfg..."
	@mv configs/TestCommand.cfg configs/TestCommand.not_embed || true
	@echo "Building..."
	@go build ${LDFLAGS} -o ${BINARY_NAME} . ; \
	EXIT_CODE=$$? ; \
	echo "Renaming TestCommand.cfg back..." ; \
	mv configs/TestCommand.not_embed configs/TestCommand.cfg || true ; \
	exit $$EXIT_CODE

test:
	go test -v -race ./...

integration-test:
	go test -v -tags=integration ./...

coverage:
	go test -coverprofile=coverage.out ./...
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
	@echo "Temporarily renaming TestCommand.cfg..."
	@mv configs/TestCommand.cfg configs/TestCommand.not_embed || true
	@echo "Building..."
	@goreleaser release --snapshot --clean ; \
	EXIT_CODE=$$? ; \
	echo "Renaming TestCommand.cfg back..." ; \
	mv configs/TestCommand.not_embed configs/TestCommand.cfg || true ; \
	exit $$EXIT_CODE

dmg:
	mkdir -p ./${BINARY_NAME}.app/Contents/MacOS
	cp ${BINARY_NAME} ./${BINARY_NAME}.app/Contents/MacOS/
	hdiutil create -volname "${BINARY_NAME}" -srcfolder ./${BINARY_NAME}.app -ov -format UDZO ${BINARY_NAME}.dmg

zip: build
	zip -r ${BINARY_NAME}.zip ${BINARY_NAME} configs

install: build
	cp ${BINARY_NAME} /usr/local/bin/

uninstall:
	rm -f /usr/local/bin/${BINARY_NAME}
