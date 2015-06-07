package ratelimiter_test

import (
	"github.com/venkat/ratelimiter"
	"testing"
	"time"
)

type HitTracker struct {
	FirstHit       time.Time
	WindowSize     time.Duration
	CurrentWindow  int
	QuotaRemaining int
	Quota          int
}

func NewHitTracker(quota int, window time.Duration) *HitTracker {
	return &HitTracker{WindowSize: window, QuotaRemaining: quota, Quota: quota}
}

//Hit simulates how the ratelimited endpoint will track the quota of calls
//and keeps tracks of that information
func (h *HitTracker) Hit() {
	if h.FirstHit.IsZero() {
		h.FirstHit = time.Now()
	}
	sinceFirstHit := time.Since(h.FirstHit)
	window := int(sinceFirstHit / h.WindowSize)
	if window != h.CurrentWindow {
		h.QuotaRemaining = h.Quota
		h.CurrentWindow = window
	}
	h.QuotaRemaining--
}

func initTest(quota int, rate time.Duration) (r *ratelimiter.RateLimiter, h *HitTracker) {
	window := time.Duration(quota) * rate
	r = ratelimiter.NewRateLimiter(quota, rate)
	h = NewHitTracker(quota, window)
	return
}

//check enforces that the ratelimiter never spends too much time throttling than
//necessary and that it never overshoots the quota of calls
func check(t *testing.T, diff time.Duration, h *HitTracker, rate time.Duration) {
	errThreshold := 60 * time.Millisecond
	if diff > errThreshold {
		t.Fatal("diff between rate and time spent throttling too high:", diff)
	}
	if h.QuotaRemaining < 0 {
		t.Fatal("hits more than quota:", h.QuotaRemaining, "window:", h.CurrentWindow, "first hit:", h.FirstHit, "since:", time.Since(h.FirstHit))
	}
}

func checkHit(t *testing.T, r *ratelimiter.RateLimiter, h *HitTracker) {
	now := time.Now()
	r.Throttle()
	r.GetThrottleChannel()
	h.Hit()
	diff := time.Since(now) - r.Rate
	check(t, diff, h, r.Rate)
}

//Test_Nowait makes continuous rate limited calls
func Test_Nowait(t *testing.T) {
	quota := int(10)
	rate := 50 * time.Millisecond
	r, h := initTest(quota, rate)
	for _ = range make([]struct{}, quota*2) {
		checkHit(t, r, h)
	}
}

func Test_Gap(t *testing.T) {
	quota := int(10)
	rate := 50 * time.Millisecond
	r, h := initTest(quota, rate)

	checkHit(t, r, h)
	for _ = range make([]struct{}, 2) {
		time.Sleep(rate * time.Duration(quota/2))

		for _ = range make([]struct{}, quota/2) {
			checkHit(t, r, h)
		}
	}
}

func Test_WaitTillWindowEnds(t *testing.T) {
	quota := int(10)
	rate := 50 * time.Millisecond
	r, h := initTest(quota, rate)

	checkHit(t, r, h)
	time.Sleep(time.Duration(quota-1) * rate)
	for _ = range make([]struct{}, quota*2) {
		checkHit(t, r, h)
	}
}
