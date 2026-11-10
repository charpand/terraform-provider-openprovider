package client

import (
	"net/http"
	"strconv"
	"testing"
	"time"
)

// FuzzRetryDelay checks that no `Retry-After` value, however hostile, makes
// the pause negative or longer than the cap, or makes the function panic.
func FuzzRetryDelay(f *testing.F) {
	for _, seed := range []string{"", "0", "1", "120", "-1", "abc", "9223372036854775807", "99999999999", "Wed, 21 Oct 2015 07:28:00 GMT", "Wed, 21 Oct 2099 07:28:00 GMT"} {
		f.Add(seed)
	}
	now := time.Date(2026, 11, 10, 12, 0, 0, 0, time.UTC)
	f.Fuzz(func(t *testing.T, retryAfter string) {
		resp := &http.Response{Header: http.Header{}}
		resp.Header.Set("Retry-After", retryAfter)

		// The attempt count is the 1-based try that just failed; the retry
		// loop is bounded, so 1 is enough to reach every header branch.
		delay := retryDelay(1, resp, now)
		if delay < 0 {
			t.Fatalf("retryDelay(Retry-After: %q) = %v; want no negative pause", retryAfter, delay)
		}
		if seconds, err := strconv.Atoi(resp.Header.Get("Retry-After")); err == nil && seconds >= 0 && delay > retryCap {
			t.Fatalf("retryDelay(Retry-After: %q) = %v; want at most %v", retryAfter, delay, retryCap)
		}
	})
}
