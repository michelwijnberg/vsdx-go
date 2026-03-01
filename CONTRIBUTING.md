# Contributing

Contributions are welcome! This project is a Go port of the Python [vsdx](https://github.com/dave-howard/vsdx) library.

## Development Setup

```bash
git clone https://github.com/MichelW6667/vsdx-go.git
cd vsdx-go
go mod download
```

## Running Tests

```bash
go test ./vsdx/... -v
```

Test fixtures are in the `tests/` directory. When adding new features, please include test cases using existing `.vsdx` test files where possible.

## Project Structure

- `vsdx/` - Main Go package with all library code
- `tests/` - Test fixture `.vsdx` files
- `python-vsdx/` - Original Python source (for reference during porting)

## Guidelines

- Follow Go conventions and idioms
- Use `github.com/beevik/etree` for XML parsing (matching the Python ElementTree approach)
- Add tests for new features
- Run `go vet ./...` and `go test ./...` before submitting

## Porting from Python

When porting a Python method, refer to the original implementation in `python-vsdx/vsdx/` and the corresponding Python tests in `tests/`. The Go API uses getter/setter methods instead of Python properties (e.g., `shape.X()` / `shape.SetX(v)` instead of `shape.x`).
