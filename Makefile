PWD=$(shell pwd)
BUILD=$(PWD)/build
VERSION=$(shell git describe --tags --dirty)

GO=go
GOFLAGS=-N -l
LDFLAGS=-X main.Version=$(VERSION)

TARGET=$(BUILD)/bin/shpp
PREFIX?=/usr

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
