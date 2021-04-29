package discovery

import (
	"fmt"
	"log"
	"net"
	"os"
)

func ServiceDiscovery(servers map[string]struct{}) {
	pc, err := net.ListenPacket("udp4", ":8829")
	if err != nil {
		log.Fatal(err)
	}
	currentHostName, err := os.Hostname()
	if err != nil {
		log.Fatal(err)
	}
	endpoint := fmt.Sprintf("http://%s:8000", currentHostName)
	defer pc.Close()
	broadcast(pc, endpoint)
	for {
		buf := make([]byte, 25)
		n, addr, err := pc.ReadFrom(buf)
		if err != nil {
			log.Printf("Broadcast read error: %s", err)
			return
		}
		server := string(buf[:n])
		if _, ok := servers[server]; !ok && server != endpoint {
			servers[string(buf[:n])] = struct{}{}
			log.Printf("Servers: %s", servers)

			_, err = pc.WriteTo([]byte(endpoint), addr)
			if err != nil {
				log.Printf("Broadcast write error: %s", err)
				return
			}
		}
	}
}

func broadcast(pc net.PacketConn, endpoint string) {
	addr, _ := net.ResolveUDPAddr("udp4", "255.255.255.255:8829")
	pc.WriteTo([]byte(endpoint), addr)
}
