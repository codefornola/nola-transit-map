package main

import (
	"context"
	"errors"
	"flag"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"golang.org/x/sync/errgroup"
)

var (
	severAddr    = flag.String("addr", ":8080", "http service address")
	severTimeout = flag.Duration("timeout", 10*time.Second, "server read and write timeouts")
)

type ServerConfig struct {
	Addr    string
	Timeout time.Duration
}

type Server struct {
	Config     ServerConfig
	Subscriber interface {
		Subscribe(context.Context, *websocket.Conn) error
	}
	Log interface {
		Printf(fomat string, v ...any)
	}
	Mux *http.ServeMux
}

func (s Server) Start(ctx context.Context) error {
	s.Mux.Handle("/", http.FileServer(http.Dir("./public")))
	// s.Mux.Handle("/public/",http.StripPrefix("/public/",fs))

	s.Mux.HandleFunc("/ws", s.newWebSocketHandler())

	log.Println("Starting server")
	server := &http.Server{
		Addr:         s.Config.Addr,
		Handler:      s.Mux,
		ReadTimeout:  s.Config.Timeout,
		WriteTimeout: s.Config.Timeout,
	}

	g, ctx := errgroup.WithContext(ctx)
	g.Go(s.ListenAndServe)
	if err := g.Wait(); err != nil {
		// TODO log
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	return server.Shutdown(ctx)
}

func (s Server) newWebSocketHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, nil)
		if err != nil {
			// TODO log the error
			s.Log.Printf("%v", err)
			return
		}
		defer conn.Close(websocket.StatusInternalError, "")

		err = s.Subscriber.Subscribe(r.Context(), conn)
		if errors.Is(err, context.Canceled) {
			// TODO log
			return
		}
		if websocket.CloseStatus(err) == websocket.StatusNormalClosure ||
			websocket.CloseStatus(err) == websocket.StatusGoingAway {
			// TODO log
			return
		}
		if err != nil {
			// TODO log
			return
		}
	}
}
