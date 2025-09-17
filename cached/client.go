package cached

import (
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"time"

	"github.com/PlakarKorp/kloset/caching"
	"golang.org/x/sys/unix"
)

// implements caching.Cache
type Cached struct {
	conn net.Conn
}

func NewClient(socket string) (*Cached, error) {
	var lockfile *os.File
	var spawned bool

	defer func() {
		if lockfile != nil {
			lockfile.Close()
			os.Remove(lockfile.Name())
		}
	}()

	attempt := 0
	for {
		conn, err := net.Dial("unix", socket)
		if err != nil {
			// windows?
			// if !errors.Is(err, syscall.ECONNREFUSED) || errors.Is(err, syscall.ENOENT) {
			// 	return nil, err
			// }

			attempt++
			if attempt > 100 {
				return nil, fmt.Errorf("failed to run the cache agent")
			}

			if lockfile == nil {
				log.Println("taking a lock on the socket")
				lockfile, err = os.OpenFile(socket+".lock", os.O_WRONLY|os.O_CREATE,
					0600)
				if err != nil {
					return nil, fmt.Errorf("failed to open lockfile: %w", err)
				}

				err := unix.Flock(int(lockfile.Fd()), unix.LOCK_EX)
				if err != nil {
					return nil, fmt.Errorf("failed to take the lock: %w", err)
				}

				/*
				 * Always retry at least once, even if we got
				 * the lock, because another client could have
				 * taken the lock, started the server and released
				 * the lock between our net.Dial and unix.Flock.
				 */
				continue
			}

			if !spawned {
				log.Println("about to spawn myself again")
				me, err := os.Executable()
				if err != nil {
					return nil, err
				}

				cached := exec.Command(me, "private-cached")
				if err := cached.Start(); err != nil {
					return nil, fmt.Errorf("failed to start cached: %w", err)
				}
				spawned = true
			}

			log.Println("connection failed; sleep to retry")
			time.Sleep(5 * time.Millisecond)
			continue
		}

		log.Println("connected!")
		return &Cached{conn}, nil
	}
}

func (c *Cached) Constructor(version, name, repoid string, opts int) (caching.Cache, error) {
	panic("unimplemented")
}

func (c *Cached) Close() error {
	return c.conn.Close()
}
