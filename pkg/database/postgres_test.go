package database

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestConnectGivesUpAfterTimeout(t *testing.T) {
	start := time.Now()
	_, err := Connect(context.Background(), "postgres://u:p@127.0.0.1:1/db?sslmode=disable", Options{
		ConnectTimeout: 1200 * time.Millisecond,
	})
	if err == nil || !strings.Contains(err.Error(), "unreachable after") {
		t.Fatalf("err = %v, want unreachable error", err)
	}
	if elapsed := time.Since(start); elapsed < time.Second || elapsed > 5*time.Second {
		t.Errorf("gave up after %v, want about the connect timeout", elapsed)
	}
}

func TestOptionsDefaults(t *testing.T) {
	o := Options{MaxOpenConns: 50}.withDefaults()
	if o.MaxOpenConns != 50 || o.MaxIdleConns != 10 || o.ConnectTimeout != 30*time.Second {
		t.Errorf("unexpected options %+v", o)
	}
}
