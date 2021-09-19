[![status][ci-status-badge]][ci-status]
[![PkgGoDev][pkg-go-dev-badge]][pkg-go-dev]

# go-problem-reporter

go-problem-reporter captures HTTP responses that conform to [RFC7807 Problem Details][rfc7807] from your web application and sends to error reporting services.

Currently supported services:

- [Sentry][]

## Synopsis

```go
package main

import (
  "github.com/getsentry/sentry-go"
  "github.com/moogar0880/problems"
  sentryhttp "github.com/getsentry/sentry-go/http"
  sentryreporter "github.com/aereal/go-problem-reporter/sentry"
)

func main() {
  h := http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
    p := problems.NewDetailedProblem(http.StatusInternalServerError, "some details"),
    problems.StatusProblemHandler(p).ServeHTTP(rw, r)
  })
  mw := sentryreporter.New(sentryreporter.Options{WaitForDelivery: true})
  withSentryHub := sentryhttp.New(sentryhttp.Options{}).Handle
  &http.Server{
    Handler: withSentryHub(mw(h)),
  }
}
```

## Installation

```sh
go get github.com/aereal/go-problem-reporter
```

## License

See LICENSE file.

[pkg-go-dev]: https://pkg.go.dev/github.com/aereal/go-problem-reporter
[pkg-go-dev-badge]: https://pkg.go.dev/badge/aereal/go-problem-reporter
[ci-status-badge]: https://github.com/aereal/go-problem-reporter/workflows/CI/badge.svg?branch=main
[ci-status]: https://github.com/aereal/go-problem-reporter/actions/workflows/CI
[rfc7807]: https://datatracker.ietf.org/doc/html/rfc7807
[Sentry]: https://sentry.io/welcome/
