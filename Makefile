PWD=$(shell pwd)
BUILD=$(PWD)/build
VERSION=$(shell git describe --tags --dirty)

GO=go
GOFLAGS=
GOFLAGS_DEBUG=-N -l
GOFLAGS_RELEASE=-trimpath -buildmode=pie
LDFLAGS=-X main.Version=$(VERSION)
LDFLAGS_RELEASE=-s -w

TARGET=$(BUILD)/bin/shpp
PREFIX?=/usr

SRCS=$(shell find src -type f -name '*.go')

FORMAT=gofmt
FORMAT_FLAGS=-w

.PHONY: release
release: GOFLAGS+=$(GOFLAGS_RELEASE)
release: LDFLAGS+=$(LDFLAGS_RELEASE)
release: $(TARGET)

.PHONY: debug
debug: GOFLAGS+=$(GOFLAGS_DEBUG)
debug: LDFLAGS+=$(LDFLAGS_DEBUG)
debug: $(TARGET)

$(TARGET): $(BUILD)/bin $(SRCS)
	$(GO) build -gcflags="$(GOFLAGS)" -ldflags="$(LDFLAGS)" -o $@ ./src

$(BUILD)/bin:
	mkdir -p $@

.PHONY: test
test:
	$(GO) test -v ./src

.PHONY: install
install: $(TARGET)
	mkdir -p $(PREFIX)/bin
	cp $(TARGET) $(PREFIX)/bin

.PHONY: clean
clean:
	rm -rf $(BUILD)

.PHONY: format
format:
	$(FORMAT) $(FORMAT_FLAGS) $(SRCS)
