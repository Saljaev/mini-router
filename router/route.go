package router

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"
)

var (
	ErrContextTimeout = errors.New("context timeout")
)

// Use add middleware to Router
func (r *Router) Use(mw HandlerFunc) {
	r.middleware = append(r.middleware, mw)
}

// POST handle new handlers func on path
func (r *Router) POST(path string, handlers ...HandlerFunc) {
	r.handle(http.MethodPost, path, handlers...)
}

// GET handle new handlers func on path
func (r *Router) GET(path string, handlers ...HandlerFunc) {
	r.handle(http.MethodGet, path, handlers...)
}

func (r *Router) handle(method, path string, handlers ...HandlerFunc) {
	if len(handlers) == 0 {
		panic("at least 1 handler")
	}

	fullPath := r.prefix + path

	parts := strings.Split(strings.Trim(fullPath, "/"), "/")
	current := r.root
	for _, part := range parts {
		if _, ok := current.children[part]; !ok {
			current.children[part] = &node{
				path:     part,
				children: make(map[string]*node),
				handler:  make(map[string][]HandlerFunc),
			}
		}
		current = current.children[part]
	}

	if len(r.middleware) > 0 {
		handlers = append(r.middleware, handlers...)
	}

	current.handler[method] = handlers
}

// ServeHTTP create APIContext and async run HandlerFunc for request path
func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	parts := strings.Split(strings.Trim(req.URL.Path, "/"), "/")
	current := r.root

	ctx := r.pool.Get().(*APIContext)
	ctx.SetCtx(w, req, r.log)
	ctx.log = ctx.log.With(slog.String("pattern", req.Method+" "+req.URL.Path))
	ctx.w.Header().Set("Content-Type", "application/json; charset=utf-8")

	defer func() {
		ctx.Wait()
		ctx.Reset()
		r.pool.Put(ctx)
	}()

	for _, part := range parts {
		if child, ok := current.children[part]; ok {
			current = child
		} else {
			ctx.Error("handler not found", errors.New("handler not found"))
			ctx.WriteFailure(http.StatusBadRequest, "bad gateway")
			return
		}
	}
	handler, exists := current.handler[req.Method]
	if !exists {
		ctx.Error("method not allowed", errors.New("method not allowed"))
		ctx.WriteFailure(http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	ctx.setHandlers(handler)
	ctx.setMiddleware(r.middleware)

	r.workersPool.Submit(ctx, handler)
}

// Shutdown graceful shutdown worker pool
func (r *Router) Shutdown() {
	r.workersPool.Shutdown()
}
