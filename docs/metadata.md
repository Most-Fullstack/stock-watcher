# Stock Watcher — Project Context
Keywords: stock-watcher, stocks, watchlist, price, ticker, REST API
Project: stock-watcher | Org: Most-Fullstack | Branch: main
## Environments
- development (default): localhost:8080
## Services
**stock-watcher** — REST API for stock watchlist management with mock price data. Repo: stock-watcher (Go). No external dependencies.
## Service Flows
**Price lookup**: Client > stock-watcher /api/stocks/:symbol > mock price generator
## Non-service Documents
- **service.md** — Full API documentation, endpoints, architecture
