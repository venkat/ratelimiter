/*
   Package ratelimiter can be used to respect the given ratelimit when calling/hitting
   some endpoint.

   To use, create a new rate limiter and call Throttle() before the method/function that
   needs to respect the ratelimit or use GetThrottleChannel() if you want to access the
   underlying channel that maintains the rate limiting
*/
package ratelimiter

import (
	"log"
	"time"
)

type nothing struct{}
type tokenBucket chan nothing

type RateLimiter struct {
	Quota          int           //no. of hits or calls that can be made in a given window
	Rate           time.Duration //rate at which you can hit or call something
	Window         time.Duration //quota * rate
	Tokens         tokenBucket   //no of calls or hits that can be made at any given time
	TokensUsed     int
	RateLimitStart time.Time
	Name           string
}

func NewRateLimiter(quota int, rate time.Duration, name string) *RateLimiter {
	r := &RateLimiter{Quota: quota, Rate: rate, Window: time.Duration(quota) * rate, Name: name}
	return r
}

func (r *RateLimiter) setup() {
	if r.Tokens == nil {
		r.Tokens = make(tokenBucket, r.Quota)
		r.RateLimitStart = time.Now()
		go r.makeTokens()
		if r.Window != 0 {
			go func() {
				for {
					time.Sleep(r.Window)
					r.reset()
				}
			}()
		}
	}
}

func (r *RateLimiter) useToken() {
	<-r.Tokens
}

//Call to Throttle should be immediately succeeded by the call to the method or function that has
//rate limiting enforced
func (r *RateLimiter) Throttle() {
	r.setup()
	r.useToken()
	r.TokensUsed++
	log.Printf("Throttling Ratelimiter %v at %v. Rate Limit %v. Rate Limit Remaining %v. Rate Limit Resets at %v. Time Remaining %v.",
		r.Name,
		time.Now(),
		r.Quota,
		r.Quota-r.TokensUsed,
		r.RateLimitStart.Add(r.Window),
		r.RateLimitStart.Add(r.Window).Sub(time.Now()))
}

//GetThrottleChannel is useful when managing multiple rate limiters that provide different
//access contexts
func (r *RateLimiter) GetThrottleChannel() tokenBucket {
	r.setup()
	return r.Tokens
}

func (r *RateLimiter) makeTokens() {
	for {
		r.Tokens <- nothing{}
		time.Sleep(r.Rate)
	}
}

func (r *RateLimiter) reset() {
	log.Printf("resetting ratelimiter %v at %v", r.Name, time.Now())
	for {
		select {
		case <-r.Tokens:
		default:
			r.RateLimitStart = time.Now()
			r.TokensUsed = 0
			return
		}
	}
}

func (r *RateLimiter) TokensLeft() int {
	r.setup()
	return len(r.Tokens)
}
