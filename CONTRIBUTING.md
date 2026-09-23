# Contributing to Faultwing

Faultwing is an early-stage learning project. Small bug fixes, documentation
improvements, tests for meaningful behavior, and focused feature proposals are
welcome.

For a larger change, open an issue first so the design and scope can be agreed
before implementation. Keep pull requests focused and explain both the user
impact and any tradeoffs.

## Local checks

Start PostgreSQL with `make db-up`, then run the checks relevant to your change:

```bash
make test
make test-integration
make test-python
make test-node
make frontend-build
make frontend-test
```

Go code should be formatted with `gofmt`. Frontend browser tests require a
local Chromium installation; see [docs/frontend.md](docs/frontend.md).

Never commit real API keys, session tokens, database credentials, customer
events, or stack traces containing private source paths. Use clearly fake
values in examples and tests.

By contributing, you agree that your work may be distributed under the
project's [MIT License](LICENSE).
