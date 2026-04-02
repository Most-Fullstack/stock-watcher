# Stock Watcher — Service Documentation

## Overview
stock-watcher is a single-process REST API written in Go using the Gin web framework. It listens for HTTP on the port defined by the PORT environment variable, defaulting to 8080 when PORT is unset. The service maintains an in-memory stock watchlist with mock price data, supports price alerts that evaluate against current prices in real time, and provides a portfolio tracker that calculates total holdings value. It does not integrate with live market feeds, databases, or message brokers. Runtime dependencies are limited to Gin for routing and middleware, plus the standard library's slog for structured logging. The codebase is a single deployable binary from the Most-Fullstack/stock-watcher repository.

## Architecture and Middleware
stock-watcher routes every request through Gin, where handlers parse paths and JSON bodies before touching shared in-memory state protected by sync.RWMutex for concurrent safety. Structured logging via slog records key events and request outcomes in a consistent machine-readable shape for aggregation and search. Graceful shutdown listens for termination signals, stops accepting new HTTP connections, allows existing requests to finish within a bounded window, then exits so orchestrators can replace instances without abrupt disconnects. The default design does not ship authentication or authorization middleware; if needed, operators typically terminate TLS and enforce identity at a reverse proxy or API gateway in front of stock-watcher.

## API Endpoints
stock-watcher exposes twelve REST operations, all over plain HTTP unless TLS is added externally.

GET /health returns a lightweight liveness response for probes and uptime checks. GET /api/stocks returns the full watchlist with current mock prices for every symbol. GET /api/stocks/:symbol returns one symbol's snapshot or a 404 when the ticker is absent from the watchlist. POST /api/stocks accepts a JSON body with a symbol field to register a new watchlist entry, normalizing the ticker to uppercase. DELETE /api/stocks/:symbol removes a symbol from the watchlist.

POST /api/alerts creates a price alert on a watched symbol. The request body requires symbol, target_price (positive number), and direction (either "above" or "below"). Returns 404 if the symbol is not already in the watchlist. GET /api/alerts returns all alerts with a triggered boolean evaluated against the current mock price at request time. DELETE /api/alerts/:id removes a specific alert by its ID.

PUT /api/portfolio/:symbol sets the share quantity for a watched symbol. The request body requires quantity (number, zero or positive). Setting quantity to zero removes the holding. Returns 404 if the symbol is not in the watchlist. GET /api/portfolio returns the portfolio summary including all holdings with current prices and per-position value, plus aggregate total_value and stock_count.

GET /api/market/summary returns an aggregate snapshot of the entire watchlist in a single request. The response includes total_symbols (count of watched tickers), average_price (mean of all current mock prices), highest_stock and lowest_stock (the symbols with the highest and lowest current prices), alert_count (total alerts), triggered_count (alerts currently triggered), and portfolio_value (sum of all holdings at current prices). This endpoint is useful for dashboard views that need a quick overview without fetching individual resources.

POST /api/stocks/:symbol/prices records an explicit price snapshot for a watched symbol and returns the captured PricePoint with price and timestamp. Returns 404 if the symbol is not in the watchlist. Prices are also recorded automatically whenever GET /api/stocks or GET /api/stocks/:symbol is called, so manual recording is optional for higher-frequency capture. GET /api/stocks/:symbol/prices returns the full price history for a symbol including all recorded data points and computed statistics: high, low, avg_price, change (absolute difference between first and last recorded price), change_pct (percentage change), and point_count. An optional limit query parameter restricts the response to the N most recent points. Returns 404 if the symbol is not in the watchlist. If no prices have been recorded yet, the response contains an empty points array and zeroed statistics.

## Data Models
Stock represents a watched symbol with two fields: Symbol (uppercase ticker string) and Price (floating-point mock quote). Prices are generated on each read using per-symbol base values with a random plus-or-minus two percent band, producing plausible market-like variation.

Alert represents a user-defined price threshold. Fields include ID (auto-generated string like "alert-1"), Symbol (the watched ticker), Target (the price threshold), Direction ("above" or "below"), Triggered (boolean evaluated at query time), and CreatedAt (timestamp when the alert was set). An alert with direction "above" triggers when the current price reaches or exceeds the target; "below" triggers when the price falls to or below the target.

Holding represents a stock position in the portfolio. Fields include Symbol (the ticker), Quantity (number of shares), Price (current mock price at query time), and Value (quantity multiplied by price, rounded to two decimal places).

Portfolio is the aggregate view returned by GET /api/portfolio. It contains Holdings (array of Holding objects), TotalValue (sum of all position values), and StockCount (number of active holdings).

MarketSummary is the aggregate snapshot returned by GET /api/market/summary. Fields include TotalSymbols (number of watched tickers), AveragePrice (mean price across all symbols), HighestStock and LowestStock (Stock objects for the current price extremes, omitted when the watchlist is empty), AlertCount (total number of alerts), TriggeredCount (alerts currently in triggered state), and PortfolioValue (total value of all holdings at current prices).

PricePoint represents a single timestamped price observation with fields Price (floating-point value) and Timestamp (RFC 3339 time).

PriceHistory is the response model for GET /api/stocks/:symbol/prices. It contains Symbol (the ticker), Points (array of PricePoint objects ordered chronologically), High (maximum recorded price), Low (minimum recorded price), AvgPrice (mean of all recorded prices), Change (absolute difference between the first and last price), ChangePct (percentage change from first to last), and PointCount (number of data points returned). History is capped at 100 points per symbol; when the cap is reached, the oldest points are evicted.

## Business Logic
stock-watcher fabricates prices in application code instead of calling external brokers. Known symbols (AAPL, GOOGL, TSLA, AMZN, MSFT) have fixed base prices; unknown symbols get a random base between 100 and 499. Each read applies a random factor between 0.98 and 1.02 to simulate price movement.

Price alerts are stored in memory alongside the watchlist. When creating an alert, the system validates that the target symbol exists in the watchlist. Alert trigger evaluation happens lazily on each GET /api/alerts request by comparing the current mock price against each alert's target and direction. This means triggered status can flip between requests as mock prices fluctuate. Deleting a stock from the watchlist does not cascade-delete its alerts, but orphaned alerts will reference a symbol with no base price.

Portfolio holdings map a watched symbol to a share quantity. Setting a holding validates the symbol is in the watchlist. Portfolio value is calculated on each GET /api/portfolio request by multiplying each holding's quantity by its current mock price. Since prices are mock and randomized, the total value will vary slightly between requests. Removing a stock from the watchlist excludes its holding from portfolio calculations until re-added.

Price history is accumulated in memory alongside other state. Each call to GET /api/stocks, GET /api/stocks/:symbol, or POST /api/stocks/:symbol/prices appends a PricePoint to the symbol's history buffer. The buffer is capped at 100 entries per symbol to bound memory usage; oldest entries are evicted when the cap is exceeded. Statistics (high, low, average, change) are computed on each read from the stored points, not pre-aggregated.

All state lives in process memory with mutex protection for concurrent access. Restarting the binary clears the watchlist, alerts, portfolio holdings, and price history.

## Troubleshooting
stock-watcher frequently fails to start when PORT is already bound by another program; freeing the port or choosing a different PORT value fixes listen errors. An empty watchlist immediately after deploy is normal for in-memory storage—use POST /api/stocks in setup scripts to seed symbols before creating alerts or setting holdings. Creating an alert or setting a holding returns 404 if you forgot to add the symbol to the watchlist first. Alert triggered status is evaluated at query time and may differ between requests due to mock price randomness. Portfolio total value fluctuates between requests for the same reason. If structured logs seem quiet, confirm the slog level and output destination match the deployment environment.

## Environment Variables
stock-watcher honors PORT as the TCP port for the HTTP server, falling back to 8080 when PORT is missing or blank so local development and many container defaults align without extra config. No database URLs, API keys, or broker secrets are required in the baseline mock implementation.
