package main

import (
	"context"
	"crypto/sha256"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/motemen/go-loghttp"
	"github.com/motemen/go-nuts/roundtime"
)

type contextKey int

var traceIDKey contextKey = 0

type requestLogger struct {
	http.ResponseWriter
	status int
}

func (l *requestLogger) Write(p []byte) (int, error) {
	return l.ResponseWriter.Write(p)
}

func (l *requestLogger) WriteHeader(status int) {
	l.status = status
	l.ResponseWriter.WriteHeader(status)
}

type serverLogger struct {
	handler http.Handler
}

func (s *serverLogger) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	timestamp := time.Now()
	id := fmt.Sprintf("%x",
		sha256.Sum256([]byte(fmt.Sprintf("%v", timestamp.UnixNano()))),
	)[:16]
	ctx := context.WithValue(r.Context(), traceIDKey, id)
	log.Printf("%s ==> %s %s", id, r.Method, r.URL)

	l := &requestLogger{w, 0}
	s.handler.ServeHTTP(l, r.WithContext(ctx))

	log.Printf("%s <== %d %s (%s)", id, l.status, r.URL,
		roundtime.Duration(time.Now().Sub(timestamp), 2))
}

func newClientLogger() *loghttp.Transport {
	return &loghttp.Transport{
		LogRequest: func(r *http.Request) {
			ctx := r.Context()
			if traceID, ok := ctx.Value(traceIDKey).(string); ok {
				log.Printf("%s --> %s %s", traceID, r.Method, r.URL)
			} else {
				panic("Attempted to log request without trace-id")
			}
		},
		LogResponse: func(r *http.Response) {
			ctx := r.Request.Context()
			var ok bool
			var start time.Time
			start, ok = ctx.Value(loghttp.ContextKeyRequestStart).(time.Time)
			if !ok {
				panic("Attempted to log response without start time")
			}
			var traceID string
			traceID, ok = ctx.Value(traceIDKey).(string)
			if !ok {
				panic("Attempted to log response without trace-id")
			}
			log.Printf("%s <-- %d %s (%s)", traceID, r.StatusCode, r.Request.URL,
				roundtime.Duration(time.Now().Sub(start), 2))
		},
	}
}
