# shpp

[![Build](https://github.com/vladaviedov/shpp/actions/workflows/build.yml/badge.svg)](https://github.com/vladaviedov/shpp/actions/workflows/build.yml)
[![Test](https://github.com/vladaviedov/shpp/actions/workflows/test.yml/badge.svg)](https://github.com/vladaviedov/shpp/actions/workflows/test.yml)

[Coverage Report](https://vladaviedov.github.io/shpp/)

Pre-processor for writing pure HTML websites.

## Requirements

- Go 1.23.
- Make.

## Build

```
make
```

## Development

- `make release` - Build release binary (same as `make`).
- `make debug` - Build debug binary.
- `make install` - Install to $PREFIX
- `make clean` - Remove build files.
- `make test` - Run tests.
- `make coverage` - Generate coverage report.
- `make format` - Reformat code.

## Documentation

- [CHANGELOG.md](./CHANGELOG.md)
- [SHPP Language Description](./docs/shpp_lang.md)
- [SHPP Directive List](./docs/directive_list.md)
