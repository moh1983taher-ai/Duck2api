package duckgo

import (
	"net/http"
	"testing"
)

func TestShouldRefreshVQD(t *testing.T) {
	tests := []struct {
		name      string
		status    int
		refreshes int
		want      bool
	}{
		{name: "refresh stale token once", status: http.StatusTeapot, refreshes: 0, want: true},
		{name: "do not refresh stale token twice", status: http.StatusTeapot, refreshes: 1, want: false},
		{name: "never retry rate limit", status: http.StatusTooManyRequests, refreshes: 0, want: false},
		{name: "do not refresh successful response", status: http.StatusOK, refreshes: 0, want: false},
		{name: "do not refresh other errors", status: http.StatusBadGateway, refreshes: 0, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := shouldRefreshVQD(tt.status, tt.refreshes); got != tt.want {
				t.Fatalf("shouldRefreshVQD(%d, %d) = %t, want %t", tt.status, tt.refreshes, got, tt.want)
			}
		})
	}
}
