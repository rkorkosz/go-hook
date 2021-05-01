package discovery

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
)

type Discovery struct {
	current string
	db      map[string]struct{}
	log     *log.Logger
}

func New(opts ...func(d *Discovery)) *Discovery {
	hostname, _ := os.Hostname()
	d := Discovery{
		current: fmt.Sprintf("http://%s:8000", hostname),
		db:      make(map[string]struct{}),
		log:     log.New(os.Stdout, "[DISCOVERY] ", log.LstdFlags),
	}
	for _, opt := range opts {
		opt(&d)
	}
	return &d
}

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

func (s *Discovery) Run(ctx context.Context) error {
	var lc net.ListenConfig
	pc, err := lc.ListenPacket(ctx, "udp4", ":8829")
	if err != nil {
		return err
	}
	defer pc.Close()
	err = broadcast(pc, s.current)
	if err != nil {
		return err
	}
	for {
		buf := make([]byte, 25)
		n, addr, err := pc.ReadFrom(buf)
		if err != nil {
			return err
		}
		server := string(buf[:n])
		if _, ok := s.db[server]; !ok && server != s.current {
			s.db[string(buf[:n])] = struct{}{}
			s.log.Printf("Servers: %s", s.db)

			_, err = pc.WriteTo([]byte(s.current), addr)
			if err != nil {
				return err
			}
		}
	}
}

func broadcast(pc net.PacketConn, endpoint string) error {
	addr, err := net.ResolveUDPAddr("udp4", "255.255.255.255:8829")
	_, err = pc.WriteTo([]byte(endpoint), addr)
	return err
}
