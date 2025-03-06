package router

import (
	"log/slog"
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
	path     string
	handler  map[string][]HandlerFunc
	children map[string]*node
}

type HandlerFunc func(ctx *APIContext)

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
