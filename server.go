package main

import (
	"context"
	"errors"
	"log"
	"net"
	"net/http"
	"time"

	"nhooyr.io/websocket"
)

type Server struct {
	Config     ServerConfig
	Subscriber interface {
		Subscribe(context.Context, WSConn) (errc <-chan error, done func())
	}
	Log *log.Logger
	Mux *http.ServeMux
}

type ServerConfig struct {
	Addr    string
	Timeout time.Duration
}

// Start attaches mux handlers and maintains the long-running server.
func (s Server) Start(ctx context.Context) error {
	s.routes(ctx)
	s.Log.Printf("INFO: starting server")
	server := &http.Server{
		Addr:         s.Config.Addr,
		Handler:      s.Mux,
		ReadTimeout:  s.Config.Timeout,
		WriteTimeout: s.Config.Timeout,
		ErrorLog:     s.Log,
		BaseContext:  func(net.Listener) context.Context { return ctx },
	}
	errc := make(chan error, 1)
	defer close(errc)
	go func() {
		errc <- server.ListenAndServe()
	}()
	select {
	case err := <-errc:
		s.Log.Printf("ERROR: server failed: %s", err)
	case <-ctx.Done():
	}

	// shutdown gracefully
	s.Log.Printf("INFO: shutting server down")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return server.Shutdown(ctx)
}

func (s Server) routes(ctx context.Context) {
	s.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./public/index.html")
	})
	s.Mux.Handle("/public/",
		http.StripPrefix("/public/", http.FileServer(http.Dir("./public"))))
	s.Mux.HandleFunc("/ws", s.newWebSocketHandler())
}

// newWebSocketHandler upgrades a request to a long-running websocket connection.
func (s Server) newWebSocketHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, nil)
		if err != nil {
			s.Log.Printf("websocket upgrade failed: %s", err)
			return
		}
		defer conn.Close(websocket.StatusInternalError, "")

		errc, done := s.Subscriber.Subscribe(r.Context(), conn)
		defer done()
		if err := <-errc; errors.Is(err, context.Canceled) ||
			websocket.CloseStatus(err) == websocket.StatusNormalClosure ||
			websocket.CloseStatus(err) == websocket.StatusGoingAway {
			s.Log.Printf("INFO: websocket subscriber disconnected: %s", err)
			return
		} else if err != nil {
			s.Log.Printf("ERROR: websocket subscriber failed: %s", err)
			return
		}
	}
}
