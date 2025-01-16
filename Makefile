PWD=$(shell pwd)
BUILD=$(PWD)/build
VERSION=$(shell git describe --tags --dirty)

GO=go
GOFLAGS=
GOFLAGS_DEBUG=-N -l
LDFLAGS=-X main.Version=$(VERSION)

TARGET=$(BUILD)/bin/shpp
PREFIX?=/usr

.PHONY: release
release: $(TARGET)

.PHONY: debug
debug: GOFLAGS+=$(GOFLAGS_DEBUG)
debug: $(TARGET)

$(TARGET): $(BUILD)/bin shpp.go
	$(GO) build -gcflags="$(GOFLAGS)" -ldflags="$(LDFLAGS)" -o $@

$(BUILD)/bin:
	mkdir -p $@

.PHONY: install
install: $(TARGET)
	mkdir -p $(PREFIX)/bin
	cp $(TARGET) $(PREFIX)/bin

.PHONY: clean
clean:
	rm -rf $(BUILD)

.PHONY: format
format:
	gofmt -w shpp.go
