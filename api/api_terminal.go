package api

import (
	"fmt"
	"log"
	"net/http"
	"os/exec"

	"github.com/PlakarKorp/kloset/appcontext"
	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/plakar/subcommands/server"
	"github.com/creack/pty"
	"github.com/gorilla/websocket"
)

func TerminalWebsocket(ctx *appcontext.AppContext, repo *repository.Repository) http.Handler {
	// Run the HTTP server in a goroutine
	go func() {
		server := &server.Server{
			ListenAddr: "127.0.0.1:9888",
			NoDelete:   true,
		}
		server.Execute(ctx, repo)
	}()

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Upgrade the HTTP connection to a WebSocket connection.
		upgrader := websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		}
		ws, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Println("Failed to upgrade to WebSocket:", err)
			return
		}
		defer ws.Close()

		cmd := exec.Command("docker", "run", "--rm", "--privileged", "-ti", "test", "bash")
		ptmx, err := pty.Start(cmd)
		if err != nil {
			fmt.Printf("Unable to start pty: %v\n", err)
			return
		}
		defer ptmx.Close()

		go func() {
			for {
				buf := make([]byte, 1024)
				n, err := ptmx.Read(buf)
				if err != nil {
					fmt.Printf("Unable to read from pty: %v\n", err)
					return
				}

				if err := ws.WriteMessage(websocket.TextMessage, buf[:n]); err != nil {
					log.Println("Error writing to WebSocket:", err)
					return
				}
			}
		}()

		go func() {
			for {
				_, message, err := ws.ReadMessage()
				fmt.Printf("Received message: %s\n", message)

				if err != nil {
					log.Println("Error reading from WebSocket:", err)
					break
				}

				n, err := ptmx.Write(message)
				if err != nil {
					fmt.Printf("Unable to write to pty: %v\n", err)
					return
				}

				fmt.Printf("written: %v\n", n)
			}
		}()

		cmd.Wait()

	})
}
