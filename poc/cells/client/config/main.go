package main

import (
	"context"
	"fmt"
	"log"

	"github.com/coreos/etcd/etcdserver/etcdserverpb"
	"google.golang.org/grpc"

	_ "github.com/ghecquet/ostau/poc/cells/client/resolver"
)

func main() {

	conn, err := grpc.Dial("cells:///etcdserverpb.KV", grpc.WithInsecure())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()

	c := etcdserverpb.NewKVClient(conn)
	r, err := c.Range(context.Background(), &etcdserverpb.RangeRequest{Key: []byte("test")})
	if err != nil {
		fmt.Println(err)
	}

	fmt.Println(r)
}
