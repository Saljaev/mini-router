package router

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
)

var (
	ErrContextTimeout = errors.New("context timeout")
)

// Use add middleware to Router
//
// # Example of usage
//
//	func main() {
//		r := router.NewRouter(slog.Default())
//		r.Use(MiddlewareLogging)
//		http.ListenAndServe("0.0.0.0:80", r)
//	}
//
//	func MiddlewareLogging(ctx *router.APIContext) {
//		ctx.Info("hello from middleware", key, data)
//	}
func (r *Router) Use(mw HandlerFunc) {
	r.middleware = append(r.middleware, mw)
}

// POST handle new handlers func on path
//
// # Example of usage
//
//	type PostReq struct {
//		Name string `json:"string"`
//	}
//
//	func (p *PostReq) IsValid() bool {
//		return len([]rune(p.Name)) > 0
//	}
//
//	func main() {
//		r := router.NewRouter(slog.Default())
//		r.POST("path_1", MainHandler)
//		r.POST("path_2", MainHandler, MinorHandler)
//		http.ListenAndServe("0.0.0.0:80", r)
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
//
//		ctx.SuccessWithData(req.Name)
//	}
//
//	func MinorHandler(ctx *router.APIContext) {
//		ctx.SuccessWithData("minor")
//	}
func (r *Router) POST(path string, handlers ...HandlerFunc) {
	r.handle(http.MethodPost, path, handlers...)
}

// GET handle new handlers func on path
//
// # Example of usage
//
//	func main() {
//		r := router.NewRouter(slog.Default())
//		r.GET("/:usertype/:id", HandlerPath)
//		r.GET("/user/", HandlerQuery)
//		http.ListenAndServe("0.0.0.0:80", r)
//	}
//
//	func HandlerPath(ctx *router.APIContext) {
//		id := ctx.GetFromPath("id")
//		userType := ctx.GetFromPath("usertype")
//		ctx.SuccessWithData(userType + " " + id)
//	}
//
//	func HandlerQuery(ctx *router.APIContext) {
//		id := ctx.GetFromQuery("id")
//		ctx.SuccessWithData(id)
//	}
func (r *Router) GET(path string, handlers ...HandlerFunc) {
	r.handle(http.MethodGet, path, handlers...)
}

func (r *Router) handle(method, path string, handlers ...HandlerFunc) {
	if len(handlers) == 0 {
		err := fmt.Errorf("0 handlers for path: %s %s", method, path)
		panic(err)
	}

	fullPath := r.prefix + path

	parts := strings.Split(strings.Trim(fullPath, "/"), "/")
	current := r.root
	for _, part := range parts {
		var isParam bool
		var paramName string

		if strings.HasPrefix(part, ":") {
			isParam = true
			paramName = part[1:]
			part = "*"
		}

		if _, ok := current.children[part]; !ok {
			current.children[part] = &node{
				path:      part,
				children:  make(map[string]*node),
				handler:   make(map[string][]HandlerFunc),
				isParam:   isParam,
				paramName: paramName,
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
	params := make(map[string]string)

	ctx := r.pool.Get().(*APIContext)
	ctx.setCtx(w, req, r.log)
	ctx.log = ctx.log.With(slog.String("pattern", req.Method+" "+req.URL.Path))
	ctx.w.Header().Set("Content-Type", "application/json; charset=utf-8")

	defer func() {
		ctx.wait()
		ctx.reset()
		r.pool.Put(ctx)
	}()

	for _, part := range parts {
		if child, ok := current.children[part]; ok {
			current = child
		} else if paramChild, ok := current.children["*"]; ok {
			current = paramChild
			params[current.paramName] = part
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

	ctx.pathParams = params
	ctx.setHandlers(handler)
	ctx.setMiddleware(r.middleware)

	r.workersPool.Submit(ctx, handler)
}

// Shutdown graceful shutdown worker pool
func (r *Router) Shutdown() {
	r.workersPool.Shutdown()
}
