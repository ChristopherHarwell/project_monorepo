# Go Build Composite Action

This is a composite GitHub Action for building Go code. It can be used in any Go repository to set up the Go environment and build the code.

## Usage

To use this action in your repository, create a workflow file (e.g., `.github/workflows/go-ci.yml`) with the following content:

```yaml
name: Go CI

on:
  push:
    branches: [ main ]
  pull_request:
    branches: [ main ]

jobs:
  build:
    name: Build
    runs-on: ubuntu-latest
    steps:
      - uses: ./.github/actions/go-build
        with:
          go-version: '1.21'  # Optional, defaults to 1.21
```

## Inputs

- `go-version`: The version of Go to use (default: '1.21')

## Requirements

This action requires:
- A Go project
- GitHub Actions enabled for your repository 