package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/ghecquet/tripr/poc/cells/index"
	"google.golang.org/grpc"

	_ "github.com/ghecquet/tripr/poc/cells/client/resolver"
)

func main() {

	conn, err := grpc.Dial("cells:///index.NodeProvider", grpc.WithInsecure())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()

	c := index.NewNodeProviderClient(conn)

	stream, err := c.ListNodes(context.Background(), &index.ListNodesRequest{})
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	for {
		msg, err := stream.Recv()
		if err != nil {
			break
		}

		fmt.Println(msg)
	}
}
