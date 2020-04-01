package main

import (
	"log"
	"net"
	"os"
	"time"

	"github.com/ghecquet/tripr/poc/cells/client/resolver"
	"github.com/ghecquet/tripr/poc/cells/index"
	"github.com/golang/protobuf/proto"
	"github.com/spf13/afero"
	"google.golang.org/grpc"
)

const (
	srvAddr       = ":0"
	discoveryAddr = "224.0.0.1:9999"
)

func main() {
	// TODO - make that a mandatory argument
	args := os.Args[1:]

	name := args[0]

	base := afero.NewOsFs()

	s := grpc.NewServer()

	lis, err := net.Listen("tcp", srvAddr)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	index.RegisterFSServer(s, index.NewHandler(base))

	go ping(name, lis.Addr(), s)

	s.Serve(lis)
}

func ping(name string, a net.Addr, s *grpc.Server) {
	addr, err := net.ResolveUDPAddr("udp", discoveryAddr)
	if err != nil {
		log.Fatal(err)
	}

	c, err := net.DialUDP("udp", nil, addr)
	for {
		data, _ := proto.Marshal(resolver.NewDNS(name))
		c.Write(data)

		for service := range s.GetServiceInfo() {
			data, _ := proto.Marshal(resolver.NewService(a.String(), service))
			c.Write(data)
		}
		time.Sleep(1 * time.Second)
	}
}
