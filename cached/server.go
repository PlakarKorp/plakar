package cached

import (
	"fmt"
	"log"
	"net"
	"sync/atomic"
	"time"
)

func ListenAndServe(socket string) error {
	listener, err := net.Listen("unix", socket)
	if err != nil {
		return fmt.Errorf("failed to bind the socket: %w", err)
	}

	// TODO: open the cache here

	var inflight atomic.Int64
	var nextID atomic.Int64
	for {
		conn, err := listener.Accept()
		if err != nil {
			// TODO: we should retry / wait and retry on
			// some errors, not everything is fatal.
			return err
		}

		inflight.Add(1)

		log.Println("accepted a connection from:", conn.LocalAddr())
		log.Println("inflight:", inflight.Load())
		go func() {
			myid := nextID.Add(1)
			defer func() {
				n := inflight.Add(-1)
				log.Println("disconnected; inflight now is", n)
				if n == 0 {
					time.Sleep(1 * time.Minute)
					if nextID.Load() == myid && inflight.Load() == 0 {
						log.Println("noone connected; shutting down")
						listener.Close()
					}
				}
			}()
			serveConn(conn)
		}()
	}
}

func serveConn(conn net.Conn) {
	for {
		var buf [1024]byte
		n, err := conn.Read(buf[:])
		if err != nil {
			log.Println("failed to read:", err)
			return
		}
		if _, err := conn.Write(buf[:n]); err != nil {
			log.Println("failed to write:", err)
			return
		}
	}
}
