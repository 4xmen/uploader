# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GORUN=$(GOCMD) run
GOTEST=$(GOCMD) test
GOCLEAN=$(GOCMD) clean
BINARY_NAME=xmen-uploader
BINARY_LINUX=$(BINARY_NAME)_linux_amd64
BINARY_MAC=$(BINARY_NAME)_darwin_amd64
BINARY_WIN=$(BINARY_NAME)_windows_amd64.exe

.PHONY: all build build-linux build-mac build-mac-arm build-windows build-all run test clean

all: clean build

# Build for current OS
build:
	$(GOBUILD) -o $(BINARY_NAME) .

# Build for Linux (amd64) - Works perfectly now thanks to pure-Go SQLite!
build-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) -o $(BINARY_LINUX) .

# Build for Mac (amd64)
build-mac:
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 $(GOBUILD) -o $(BINARY_MAC) .

# Build for Mac (Apple Silicon / ARM64)
build-mac-arm:
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 $(GOBUILD) -o $(BINARY_NAME)_darwin_arm64 .

# Build for Windows (amd64)
build-windows:
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 $(GOBUILD) -o $(BINARY_WIN) .

# Build all common platforms
build-all: build-linux build-mac build-mac-arm build-windows

# Run the application locally
run:
	$(GORUN) .

# Run the test suite
test:
	$(GOTEST) ./...

# Clean build artifacts
clean:
	$(GOCLEAN)
	rm -f $(BINARY_NAME)
	rm -f $(BINARY_LINUX)
	rm -f $(BINARY_MAC)
	rm -f $(BINARY_NAME)_darwin_arm64
	rm -f $(BINARY_WIN)
	rm -rf dist
