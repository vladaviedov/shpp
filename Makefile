PWD=$(shell pwd)
BUILD=$(PWD)/build

GO=go

TARGET=$(BUILD)/bin/shpp

$(TARGET): $(BUILD)/bin shpp.go
	$(GO) build -o $@

$(BUILD)/bin:
	mkdir -p $@

.PHONY: clean
clean:
	rm -rf $(BUILD)
