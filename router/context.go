package router

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"sync"
	"time"
)

type (
	APIContext struct {
		w          http.ResponseWriter
		r          *http.Request
		ctx        context.Context
		cancel     context.CancelFunc
		pathParams map[string]string
		handlers   []HandlerFunc
		log        *slog.Logger

		wg *sync.WaitGroup
	}

	validator interface {
		IsValid() bool
	}

	Error struct {
		ErrorMessage string `json:"errors"`
	}
)

var (
	ErrGetBodyFromGet   = errors.New("get body request from get method")
	ErrGetPathFromPost  = errors.New("get path from post method")
	ErrGetQueryFromPost = errors.New("get query from post method")
)

// check for implementation
var _ context.Context = (*APIContext)(nil)

func (ctx *APIContext) setCtx(
	w http.ResponseWriter,
	r *http.Request,
	l *slog.Logger,
) {
	ctx.w = w
	ctx.r = r
	ctx.log = l
	ctx.handlers = nil
	ctx.pathParams = make(map[string]string)
	ctx.wg = &sync.WaitGroup{}
	ctx.withCancel()
}

func (ctx *APIContext) setHandlers(h []HandlerFunc) {
	ctx.handlers = h
}

func (ctx *APIContext) setMiddleware(mw []HandlerFunc) {
	ctx.handlers = append(mw, ctx.handlers...)
}

// Decode parsing request body to your struct dest,
// which must have method IsValid() bool.
// Method IsValid() validate body of request with your
// conditions
//
// # Example of usage
//
//	type Request struct {
//		msg string `json:"msg"`
//	}
//
//	func (p *PostReq) IsValid() bool {
//		l := len([]rune(p.msg))
//		return l > 0 && l < 25
//	}
//
//	func MainHandler(ctx *router.APIContext) {
//		var req PostReq
//
//		err := ctx.Decode(&req)
//		if err != nil {
//			ctx.Error("failed to decode req", err)
//			ctx.WriteFailure(http.StatusBadRequest, "invalid request")
//			return
//		}
//	}
func (ctx *APIContext) Decode(dest validator) error {
	if ctx.r.Method == http.MethodGet {
		return ErrGetBodyFromGet
	}

	defer ctx.r.Body.Close()

	err := json.NewDecoder(ctx.r.Body).Decode(&dest)
	if err != nil || !dest.IsValid() {
		if err == nil {
			err = errors.New("invalid request")
		}

		return err
	}

	return nil
}

// WriteFailure write to response writer code and struct
// Error with msg
func (ctx *APIContext) WriteFailure(code int, msg string) {
	ctx.w.WriteHeader(code)

	data, err := json.Marshal(Error{ErrorMessage: msg})
	if err != nil {
		ctx.Error("json.Marshal error", err)
	}

	_, err = ctx.w.Write(data)
	if err != nil {
		ctx.Error("response write error", err)
	}
	ctx.cancel()
}

// SuccessWithData write to response writer status OK and
// marshalled to JSON data
func (ctx *APIContext) SuccessWithData(data interface{}) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		ctx.Error("failed to json.Marhal", err)
	}

	ctx.w.WriteHeader(http.StatusOK)
	_, err = ctx.w.Write(jsonData)
	if err != nil {
		ctx.Error("failed to write to response writer", err)
	}
}

func (ctx *APIContext) Error(msg string, err error) {
	ctx.log.Error(msg, slog.Any("error", err))
}

func (ctx *APIContext) Debug(msg, key string, value any) {
	ctx.log.Debug(msg, slog.Any(key, value))
}

func (ctx *APIContext) Info(msg, key string, value any) {
	ctx.log.Info(msg, slog.Any(key, value))
}

func (ctx *APIContext) reset() {
	ctx.w = nil
	ctx.r = nil
	ctx.handlers = nil
	ctx.log = nil
	ctx.ctx = context.Background()
	ctx.wg = &sync.WaitGroup{}
	ctx.pathParams = nil
}

func (ctx *APIContext) wait() {
	ctx.wg.Wait()
}

func (ctx *APIContext) Done() <-chan struct{} {
	return ctx.ctx.Done()
}

func (ctx *APIContext) WithTimeout(d time.Duration) {
	ctxTimeout, cancel := context.WithTimeout(ctx.ctx, d)

	ctx.ctx = ctxTimeout
	ctx.cancel = cancel
}

func (ctx *APIContext) withCancel() {
	ctxWithCancel, cancel := context.WithCancel(context.Background())
	ctx.ctx = ctxWithCancel
	ctx.cancel = cancel
}

func (ctx *APIContext) Deadline() (d time.Time, ok bool) {
	return ctx.ctx.Deadline()
}

func (ctx *APIContext) Err() error {
	return ctx.ctx.Err()
}

func (ctx *APIContext) Value(key any) any {
	return ctx.ctx.Value(key)
}

func (ctx *APIContext) Set(key, value any) {
	ctx.ctx = context.WithValue(ctx.ctx, key, value)
}

func (ctx *APIContext) GetFromHeader(key string) string {
	return ctx.r.Header.Get(key)
}

func (ctx *APIContext) GetFromQuery(key string) (string, error) {
	if ctx.r.Method == http.MethodGet {
		return ctx.r.URL.Query().Get(key), nil
	}

	return "", ErrGetQueryFromPost
}

func (ctx *APIContext) GetFromPath(key string) (string, error) {
	if ctx.r.Method == http.MethodGet {
		return ctx.pathParams[key], nil
	}

	return "", ErrGetPathFromPost
}
