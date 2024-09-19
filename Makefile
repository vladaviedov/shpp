PWD=$(shell pwd)
BUILD=$(PWD)/build
# VERSION=$(shell git describe --tags --dirty)
VERSION=pre0.1-g$(shell git rev-parse --short HEAD)$(shell git diff-index --quiet HEAD || echo -dirty)

GO=go
LDFLAGS=-X main.Version=$(VERSION)

TARGET=$(BUILD)/bin/shpp

$(TARGET): $(BUILD)/bin shpp.go
	$(GO) build -ldflags="$(LDFLAGS)" -o $@

$(BUILD)/bin:
	mkdir -p $@

.PHONY: clean
clean:
	rm -rf $(BUILD)
