BINARY_NAME = rainbow-roads
VERSION := $(shell git describe --exact-match --tags 2>/dev/null)
LDFLAGS = -ldflags "-X main.Version=$(VERSION)"

build:
	GOARCH=amd64 GOOS=darwin  go build -o bin/darwin/${BINARY_NAME}      $(LDFLAGS) .
	GOARCH=amd64 GOOS=linux   go build -o bin/linux/${BINARY_NAME}       $(LDFLAGS) .
	GOARCH=amd64 GOOS=windows go build -o bin/windows/${BINARY_NAME}.exe $(LDFLAGS) .

release: build
	mkdir -p release
	tar -acf release/$(BINARY_NAME)-$(VERSION)-darwin-amd64.tar.gz README.md LICENSE -C bin/darwin ${BINARY_NAME}
	tar -acf release/$(BINARY_NAME)-$(VERSION)-linux-amd64.tar.gz  README.md LICENSE -C bin/linux  ${BINARY_NAME}
	zip -rj  release/$(BINARY_NAME)-$(VERSION)-windows-amd64.zip   README.md LICENSE bin/windows/${BINARY_NAME}.exe
