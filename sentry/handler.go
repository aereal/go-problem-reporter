package sentry

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/aereal/gomas/httputil"
	sentrysdk "github.com/getsentry/sentry-go"
	"github.com/moogar0880/problems"
)

var contentTypeJSON = "application/json"

func isValidContentType(ct string) bool {
	return ct == contentTypeJSON || ct == problems.ProblemMediaType
}

func decodeJSON(r io.Reader) (*problems.DefaultProblem, error) {
	var p problems.DefaultProblem
	err := json.NewDecoder(r).Decode(&p)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// Options configure a middleware.
type Options struct {
	// WaitForDelivery indicates whether to block the current goroutine and wait until the event has been reported to Sentry.
	WaitForDelivery bool

	// FlushTimeout for the delivery events. Defaults to 2s. Only relevant when WaitForDelivery is true.
	FlushTimeout time.Duration
}

// New returns new middleware that reports problems to Sentry.
//
// The problems must be conform to [RFC-7807](https://datatracker.ietf.org/doc/html/rfc7807).
//
// The request's context must hold *sentry.Hub such as using [sentry-go/http](https://github.com/getsentry/sentry-go/tree/master/http).
func New(opts Options) httputil.Middleware {
	timeout := opts.FlushTimeout
	if timeout == 0 {
		timeout = time.Second * 2
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
			buf := new(bytes.Buffer)
			tw := httputil.NewTeeResponseWriter(rw, buf)
			next.ServeHTTP(tw, r)
			if !isValidContentType(tw.Header().Get("content-type")) {
				return
			}
			hub := sentrysdk.GetHubFromContext(r.Context())
			if hub == nil {
				return
			}
			p, err := decodeJSON(buf)
			if err != nil {
				return
			}
			hub.WithScope(func(scope *sentrysdk.Scope) {
				msg := p.Title
				if p.Detail != "" {
					scope.SetExtra("problem.detail", p.Detail)
					msg = p.Detail
				}
				if p.Status != 0 {
					scope.SetExtra("problem.status", p.Status)
				}
				if p.Instance != "" {
					scope.SetExtra("problem.instance", p.Instance)
				}
				_ = hub.CaptureMessage(msg)
				if opts.WaitForDelivery {
					_ = hub.Flush(timeout)
				}
			})
		})
	}
}
