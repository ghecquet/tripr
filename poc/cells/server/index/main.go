package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"time"

	"github.com/ghecquet/tripr/poc/cells/index"
	"github.com/spf13/afero"
	"google.golang.org/grpc"
)

const (
	srvAddr       = ":8100"
	discoveryAddr = "224.0.0.1:9999"
)

func main() {
	base := afero.NewOsFs()
	cache := afero.NewMemMapFs()

	// Initial sync
	afero.Walk(base, "/Library", func(path string, fi os.FileInfo, err error) error {
		cache.Create(path)

		return nil
	})

	s := grpc.NewServer()

	lis, err := net.Listen("tcp", srvAddr)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	fmt.Println(lis.Addr())

	index.RegisterNodeProviderServer(s, index.NewHandler(cache))

	go ping(lis.Addr(), s)

	s.Serve(lis)
}

func ping(a net.Addr, s *grpc.Server) {
	addr, err := net.ResolveUDPAddr("udp", discoveryAddr)
	if err != nil {
		log.Fatal(err)
	}

	c, err := net.DialUDP("udp", nil, addr)
	for {
		for service := range s.GetServiceInfo() {
			fmt.Println("Ping sent ", a.String()+","+service)
			c.Write([]byte(a.String() + "," + service))
		}
		time.Sleep(1 * time.Second)
	}
}
