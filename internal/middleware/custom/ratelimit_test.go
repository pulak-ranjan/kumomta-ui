package custom

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"golang.org/x/time/rate"
)

func TestRateLimiter(t *testing.T) {
	// 2 req/sec
	limiter := NewRateLimiter(rate.Every(time.Second), 2)
	handler := limiter.Limit(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req, _ := http.NewRequest("GET", "/", nil)
	req.RemoteAddr = "192.168.1.1:12345"

	// 1st request - should pass
	rr1 := httptest.NewRecorder()
	handler.ServeHTTP(rr1, req)
	if rr1.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr1.Code)
	}

	// 2nd request - should pass
	rr2 := httptest.NewRecorder()
	handler.ServeHTTP(rr2, req)
	if rr2.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr2.Code)
	}

	// 3rd request - should fail (exceeded burst of 2)
	rr3 := httptest.NewRecorder()
	handler.ServeHTTP(rr3, req)
	if rr3.Code != http.StatusTooManyRequests {
		t.Errorf("expected 429, got %d", rr3.Code)
	}

	// Test different port same IP - should also fail (visitor tracked by IP)
	req2, _ := http.NewRequest("GET", "/", nil)
	req2.RemoteAddr = "192.168.1.1:54321"
	rr4 := httptest.NewRecorder()
	handler.ServeHTTP(rr4, req2)
	if rr4.Code != http.StatusTooManyRequests {
		t.Errorf("expected 429 for different port same IP, got %d", rr4.Code)
	}

	// Test different IP - should pass
	req3, _ := http.NewRequest("GET", "/", nil)
	req3.RemoteAddr = "192.168.1.2:12345"
	rr5 := httptest.NewRecorder()
	handler.ServeHTTP(rr5, req3)
	if rr5.Code != http.StatusOK {
		t.Errorf("expected 200 for different IP, got %d", rr5.Code)
	}
}
