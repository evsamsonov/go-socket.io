package socketio

import (
	"net/http"
	"sync"

	engineio "github.com/googollee/go-engine.io"
)

type namespaceHandlers struct {
	mu       sync.RWMutex
	handlers map[string]*namespaceHandler
}

func newNamespaceHandlers() *namespaceHandlers {
	return &namespaceHandlers{
		handlers: make(map[string]*namespaceHandler),
	}
}

func (h *namespaceHandlers) Set(namespace string, handler *namespaceHandler) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.handlers[namespace] = handler
}

func (h *namespaceHandlers) Get(nsp string) (*namespaceHandler, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	handler, ok := h.handlers[nsp]
	return handler, ok
}

// Server is a go-socket.io server.
type Server struct {
	handlers *namespaceHandlers
	eio      *engineio.Server
}

// NewServer returns a server.
func NewServer(c *engineio.Options) (*Server, error) {
	eio, err := engineio.NewServer(c)
	if err != nil {
		return nil, err
	}
	return &Server{
		handlers: newNamespaceHandlers(),
		eio:      eio,
	}, nil
}

// Close closes server.
func (s *Server) Close() error {
	return s.eio.Close()
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.eio.ServeHTTP(w, r)
}

// OnConnect set a handler function f to handle open event for
// namespace nsp.
func (s *Server) OnConnect(nsp string, f func(Conn) error) {
	h := s.getNamespace(nsp)
	h.OnConnect(f)
}

// OnDisconnect set a handler function f to handle disconnect event for
// namespace nsp.
func (s *Server) OnDisconnect(nsp string, f func(Conn, string)) {
	h := s.getNamespace(nsp)
	h.OnDisconnect(f)
}

// OnError set a handler function f to handle error for namespace nsp.
func (s *Server) OnError(nsp string, f func(error)) {
	h := s.getNamespace(nsp)
	h.OnError(f)
}

// OnEvent set a handler function f to handle event for namespace nsp.
func (s *Server) OnEvent(nsp, event string, f interface{}) {
	h := s.getNamespace(nsp)
	h.OnEvent(event, f)
}

// Serve serves go-socket.io server
func (s *Server) Serve() error {
	for {
		conn, err := s.eio.Accept()
		if err != nil {
			return err
		}
		go s.serveConn(conn)
	}
}

func (s *Server) serveConn(c engineio.Conn) {
	_, err := newConn(c, s.handlers)
	if err != nil {
		root, _ := s.handlers.Get("")
		if root != nil && root.onError != nil {
			root.onError(err)
		}
		return
	}
}

func (s *Server) getNamespace(nsp string) *namespaceHandler {
	if nsp == "/" {
		nsp = ""
	}
	ret, ok := s.handlers.Get(nsp)
	if ok {
		return ret
	}
	handler := newHandler()
	s.handlers.Set(nsp, handler)
	return handler
}
