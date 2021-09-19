package sentry

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/aereal/gomas/httputil"
	"github.com/getsentry/sentry-go"
	sentrysdk "github.com/getsentry/sentry-go"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/moogar0880/problems"
)

func Test_ok(t *testing.T) {
	withInstanceProblem := problems.NewStatusProblem(http.StatusInternalServerError)
	withInstanceProblem.Instance = "http://instance.example/"

	testCases := []struct {
		name       string
		problem    *problems.DefaultProblem
		wantEvents []*sentrysdk.Event
	}{
		{
			"minimal",
			problems.NewStatusProblem(http.StatusInternalServerError),
			[]*sentrysdk.Event{
				{
					Level:   sentrysdk.LevelInfo,
					Message: http.StatusText(http.StatusInternalServerError),
					Extra: map[string]interface{}{
						"problem.status": http.StatusInternalServerError,
					},
				},
			},
		},
		{
			"with detail",
			problems.NewDetailedProblem(http.StatusInternalServerError, "some details"),
			[]*sentrysdk.Event{
				{
					Level:   sentrysdk.LevelInfo,
					Message: "some details",
					Extra: map[string]interface{}{
						"problem.status": http.StatusInternalServerError,
						"problem.detail": "some details",
					},
				},
			},
		},
		{
			"with instance",
			withInstanceProblem,
			[]*sentrysdk.Event{
				{
					Level:   sentrysdk.LevelInfo,
					Message: http.StatusText(http.StatusInternalServerError),
					Extra: map[string]interface{}{
						"problem.status":   http.StatusInternalServerError,
						"problem.instance": "http://instance.example/",
					},
				},
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			h := http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
				if !sentry.HasHubOnContext(r.Context()) {
					t.Errorf("request context has no hub")
				}
				problems.StatusProblemHandler(tc.problem).ServeHTTP(rw, r)
			})
			srv := httptest.NewServer(withSentryHub()(New(Options{WaitForDelivery: true})(h)))
			defer srv.Close()

			var mux sync.Mutex
			var gotEvents []*sentry.Event
			err := sentry.Init(sentry.ClientOptions{
				BeforeSend: func(event *sentry.Event, hint *sentry.EventHint) *sentry.Event {
					mux.Lock()
					defer mux.Unlock()
					gotEvents = append(gotEvents, event)
					return event
				},
			})
			if err != nil {
				t.Fatal(err)
			}

			resp, err := srv.Client().Get(srv.URL)
			if err != nil {
				t.Fatal(err)
			}
			if !sentry.Flush(time.Second * 5) {
				t.Fatal("timeout sentry.Flush()")
			}
			if resp.StatusCode != tc.problem.Status {
				t.Errorf("status: want=%d got=%d", tc.problem.Status, resp.StatusCode)
			}
			if diff := cmp.Diff(tc.wantEvents, gotEvents, sentryEventCmpOptions); diff != "" {
				t.Errorf("Events (-want, +got):\n%s", diff)
			}
		})
	}
}

func Test_noSentryHub(t *testing.T) {
	wantStatus := http.StatusInternalServerError
	h := http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		p := problems.NewStatusProblem(wantStatus)
		problems.StatusProblemHandler(p).ServeHTTP(rw, r)
	})
	srv := httptest.NewServer(New(Options{WaitForDelivery: true})(h))
	defer srv.Close()

	resp, err := srv.Client().Get(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != wantStatus {
		t.Errorf("status: want=%d got=%d", wantStatus, resp.StatusCode)
	}
}

func withSentryHub() httputil.Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
			hub := sentry.CurrentHub().Clone()
			ctx := sentry.SetHubOnContext(r.Context(), hub)
			next.ServeHTTP(rw, r.WithContext(ctx))
		})
	}
}

var sentryEventCmpOptions = cmp.Options{
	cmpopts.IgnoreFields(
		sentry.Event{},
		"Contexts", "EventID", "Platform", "Release", "Sdk", "ServerName", "Tags", "Timestamp",
	),
	cmpopts.IgnoreFields(
		sentry.Request{},
		"Env",
	),
}
