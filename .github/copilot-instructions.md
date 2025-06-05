We use `gopkg.in/check.v1` for our tests. This package should always be imported with no alias added (no dot import as well).

Always run `golangci-lint run --timeout=10m` after you change the code to make sure the code is in compliance with our code style standards.