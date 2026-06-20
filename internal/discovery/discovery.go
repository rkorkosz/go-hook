// Package discovery provides server discovery over UDP.
package discovery

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"syscall"
	"time"

	"golang.org/x/sys/unix"
)

// Discovery holds all discovered servers
type Discovery struct {
	Current      string
	db           map[string]struct{}
	Log          *log.Logger
	ListenConfig net.ListenConfig
}

// New creates discovery object
func New(opts ...func(d *Discovery)) *Discovery {
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown"
	}
	d := Discovery{
		Current: fmt.Sprintf("http://%s:8000", hostname),
		db:      make(map[string]struct{}),
		Log:     log.New(os.Stdout, "[DISCOVERY] ", log.LstdFlags),
		ListenConfig: net.ListenConfig{
			Control: func(network, address string, c syscall.RawConn) error {
				var opErr error
				err := c.Control(func(fd uintptr) {
					opErr = unix.SetsockoptInt(int(fd), unix.SOL_SOCKET, unix.SO_REUSEPORT, 1)
				})
				if err != nil {
					return err
				}
				return opErr
			},
		},
	}
	for _, opt := range opts {
		opt(&d)
	}
	return &d
}

// Iter creates an iterator that iterates over all the servers in db
func (s *Discovery) Iter() chan string {
	out := make(chan string)
	go func(out chan string) {
		for srv := range s.db {
			out <- srv
		}
		close(out)
	}(out)
	return out
}

// Run creates a loop that performs a server discovery
func (s *Discovery) Run(ctx context.Context) error {
	pc, err := s.ListenConfig.ListenPacket(ctx, "udp", ":8829")
	if err != nil {
		return err
	}
	defer func() {
		if err := pc.Close(); err != nil {
			s.Log.Printf("Error closing packet conn: %v", err)
		}
	}()
	addr, err := net.ResolveUDPAddr("udp", "255.255.255.255:8829")
	if err != nil {
		return err
	}
	_, err = pc.WriteTo([]byte(s.Current), addr)
	if err != nil {
		return err
	}
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		buf := make([]byte, 50)
		// Set a read deadline so the loop can check ctx periodically.
		if err := pc.SetReadDeadline(time.Now().Add(time.Second)); err != nil {
			s.Log.Printf("Error setting read deadline: %v, retrying...", err)
		}

		n, _, err := pc.ReadFrom(buf)
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				// Timeout from SetReadDeadline — loop back to check ctx.
				continue
			}
			if ctx.Err() != nil {
				return nil
			}
			// Transient network error — wait and retry.
			s.Log.Printf("Discovery read error: %v, retrying...", err)
			select {
			case <-ctx.Done():
				return nil
			case <-time.After(500 * time.Millisecond):
			}
			continue
		}

		server := string(buf[:n])
		if _, ok := s.db[server]; !ok && server != s.Current {
			s.db[string(buf[:n])] = struct{}{}

			// Build a human-readable server list for logging.
			var servers []string
			for srv := range s.db {
				servers = append(servers, srv)
			}
			s.Log.Printf("Servers: %s", strings.Join(servers, ", "))

			_, err = pc.WriteTo([]byte(s.Current), addr)
			if err != nil {
				return err
			}
		}
	}
}
