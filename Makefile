PREFIX ?= /usr
BINARY_NAME=dynomark

all: build

build:
	go build -o ${BINARY_NAME} .

run:
	go build -o ${BINARY_NAME} .
	./${BINARY_NAME}

install:
	@install -Dm755 ${BINARY_NAME} $(DESTDIR)$(PREFIX)/bin/${BINARY_NAME}

uninstall:
	@rm -f $(DESTDIR)$(PREFIX)/bin/${BINARY_NAME}

clean:
	go clean
	rm ${BINARY_NAME}

