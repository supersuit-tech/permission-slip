package main

import (
	"context"
	"net/http"
	"time"

	"github.com/supersuit-tech/permission-slip-web/api"
	"github.com/supersuit-tech/permission-slip-web/db"
)

// handleHealth returns a handler that reports server health, including
// database connectivity status when a database connection is configured.
//
// Responses:
//   - 200 {"status":"ok","database":"ok"} — healthy with DB
//   - 200 {"status":"ok"} — healthy without DB
//   - 503 {"status":"degraded","database":"unreachable"} — DB unreachable
func handleHealth(d db.DBTX) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]string{"status": "ok"}

		if d != nil {
			ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
			defer cancel()

			var n int
			err := d.QueryRow(ctx, "SELECT 1").Scan(&n)
			if err != nil {
				resp["status"] = "degraded"
				resp["database"] = "unreachable"
				api.RespondJSON(w, http.StatusServiceUnavailable, resp)
				return
			}
			resp["database"] = "ok"
		}

		api.RespondJSON(w, http.StatusOK, resp)
	}
}
