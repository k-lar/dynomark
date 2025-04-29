PREFIX ?= /usr
BINARY_NAME=dynomark

all: build

build:
	go build -o ${BINARY_NAME} .

build-macos:
	GOOS=darwin GOARCH=amd64 go build -o ${BINARY_NAME} .

build-windows:
	GOOS=windows GOARCH=amd64 go build -o ${BINARY_NAME}.exe .

test:
	go test -v ./...

run:
	go build -o ${BINARY_NAME} .
	./${BINARY_NAME}

install:
	@# Create the bin directory if it doesn't exist (Mac doesn't support install -D)
	@if [ ! -d $(DESTDIR)$(PREFIX)/bin ]; then \
		mkdir -p $(DESTDIR)$(PREFIX)/bin; \
	fi

	@install -m755 ${BINARY_NAME} $(DESTDIR)$(PREFIX)/bin/${BINARY_NAME}

uninstall:
	@rm -f $(DESTDIR)$(PREFIX)/bin/${BINARY_NAME}

clean:
	go clean
	rm ${BINARY_NAME}

