package router

import (
	"log/slog"
	"net/http"
	"sync"
)

const CountOfWorkers = 1000
const SizeOfQueue = 5000

// Router single entity with table of route
type Router struct {
	root        *node
	middleware  []HandlerFunc
	pool        sync.Pool
	log         *slog.Logger
	prefix      string
	parent      *Router
	workersPool *WorkerPool
}

type node struct {
	path      string
	handler   map[string][]HandlerFunc
	children  map[string]*node
	isParam   bool   // flag for param in path
	paramName string // name for param in path
}

// HandlerFunc is a singe handler with APIContext
type HandlerFunc func(ctx *APIContext)

// WrapHandler is a wrapper for default handler like
// func (w http.ResponseWriter, r *http.Request)
func WrapHandler(handler func(w http.ResponseWriter, r *http.Request)) HandlerFunc {
	return func(ctx *APIContext) {
		handler(ctx.w, ctx.r)
	}
}

// NewRouter
//
// # Example of usage
//
//	func main() {
//		r := router.NewRouter(slog.Default())
//		r.Use(MiddlewareLogging)
//		r.POST("/", MainHandler)
//		http.ListenAndServe("0.0.0.0:80", r)
//	}
//
//	func MiddlewareLogging(ctx *router.APIContext) {
//		ctx.Info("start handling", "handler", 1)
//		ctx.Set("idx", 2)
//	}
//
//	type Resp struct {
//		Msg string `json:"msg"`
//	}
//
//	func MainHandler(ctx *router.APIContext) {
//		mainResp := Resp{
//			Msg: "hello world",
//		}
//
//		handlerIndex := ctx.Value("idx")
//		if handlerIndex == nil {
//			ctx.Error("fail to get handler index", errors.New("ctx get value error"))
//			ctx.WriteFailure(http.StatusInternalServerError, "internal error")
//			return
//		}
//
//		ctx.Info("stop handling", "handler", handlerIndex)
//		ctx.SuccessWithData(mainResp)
//	}
func NewRouter(log *slog.Logger) *Router {
	return &Router{
		root: &node{children: make(map[string]*node)},
		pool: sync.Pool{
			New: func() interface{} { return &APIContext{} },
		},
		log:         log,
		parent:      nil,
		workersPool: NewWorkerPool(CountOfWorkers, SizeOfQueue),
	}
}

// Group — create router group with own middleware
// and inheritance parent middleware
func (r *Router) Group(prefix string) *Router {
	return &Router{
		root:       r.root,
		middleware: r.middleware,
		pool: sync.Pool{
			New: func() interface{} { return &APIContext{} },
		},
		log:         r.log,
		prefix:      r.prefix + prefix,
		parent:      r,
		workersPool: r.workersPool,
	}
}
