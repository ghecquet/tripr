package main

import (
	"fmt"
	"log"
	"net"
	"time"

	"github.com/coreos/etcd/etcdserver/etcdserverpb"
	"google.golang.org/grpc"
)

const (
	srvAddr       = ":0"
	discoveryAddr = "224.0.0.1:9999"
)

func main() {
	s := grpc.NewServer()

	lis, err := net.Listen("tcp", srvAddr)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	fmt.Println(lis.Addr())

	etcdserverpb.RegisterKVServer(s, &Handler{})

	go ping(lis.Addr(), s)

	s.Serve(lis)

	// Checkout hashicorp plugin ??
}

func ping(a net.Addr, s *grpc.Server) {
	addr, err := net.ResolveUDPAddr("udp", discoveryAddr)
	if err != nil {
		log.Fatal(err)
	}

	c, err := net.DialUDP("udp", nil, addr)
	for {
		for service := range s.GetServiceInfo() {
			c.Write([]byte(a.String() + "," + service))
		}
		time.Sleep(1 * time.Second)
	}
}
