# Go Tests Composite Action

This is a composite GitHub Action for running Go unit tests. It can be used in any Go repository to set up the Go environment and run tests.

## Usage

To use this action in your repository, create a workflow file (e.g., `.github/workflows/go-tests.yml`) with the following content:

```yaml
name: Go Tests

on:
  push:
    branches: [ main ]
  pull_request:
    branches: [ main ]

jobs:
  test:
    name: Run Unit Tests
    runs-on: ubuntu-latest
    steps:
      - uses: ./.github/actions/go-tests
        with:
          go-version: '1.21'  # Optional, defaults to 1.21
```

## Inputs

- `go-version`: The version of Go to use (default: '1.21')

## Requirements

This action requires:
- A Go project with tests
- GitHub Actions enabled for your repository 