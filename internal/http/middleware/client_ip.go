package middleware

import "github.com/labstack/echo/v4"

// ClientIP reads X-Forwarded-For only behind a trusted proxy. Otherwise any
// client could set the header and pick the IP that logs and rate limits see.
// Even then, Echo honours the header only when the direct peer is on a
// loopback or private network, where a real proxy sits.
func ClientIP(trustProxy bool) echo.IPExtractor {
	if trustProxy {
		return echo.ExtractIPFromXFFHeader()
	}

	return echo.ExtractIPDirect()
}
