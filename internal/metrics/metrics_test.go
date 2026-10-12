package metrics

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func scrape(t *testing.T, m *Metrics) string {
	t.Helper()
	rec := httptest.NewRecorder()
	m.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	body, _ := io.ReadAll(rec.Body)
	return string(body)
}

func TestMetricsExposition(t *testing.T) {
	m := New("1.2.3")

	m.RequestStarted()
	m.RequestFinished("POST", "/api/transfers", 201, 30*time.Millisecond)
	m.Operation("transfer", 2500, ResultOK)
	m.Operation("transfer", 9999, "INSUFFICIENT_BALANCE")

	out := scrape(t, m)
	for _, want := range []string{
		`bank_build_info{go_version="go`,
		`version="1.2.3"} 1`,
		`http_requests_total{method="POST",route="/api/transfers",status="201"} 1`,
		`http_request_duration_seconds_bucket{method="POST",route="/api/transfers",le="0.05"} 1`,
		`http_requests_in_flight 0`,
		`bank_operations_total{operation="transfer",result="ok"} 1`,
		`bank_operations_total{operation="transfer",result="INSUFFICIENT_BALANCE"} 1`,
		`bank_operation_amount_cents_total{operation="transfer"} 2500`,
		`go_goroutines`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q", want)
		}
	}
}
