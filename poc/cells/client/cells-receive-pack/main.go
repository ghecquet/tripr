package main

import (
	"context"
	"log"
	"os"
	"path/filepath"

	"github.com/ghecquet/tripr/poc/cells/billyfs"
	"github.com/spf13/afero"
	"gopkg.in/src-d/go-git.v4/plumbing/protocol/packp"
	"gopkg.in/src-d/go-git.v4/plumbing/transport"
	"gopkg.in/src-d/go-git.v4/plumbing/transport/server"
	"gopkg.in/src-d/go-git.v4/utils/ioutil"
)

func main() {
	gitDir, err := filepath.Abs(os.Args[1] + "/.git")
	if err != nil {
		log.Fatal(err)
	}

	// Loading filesystem directly
	loader := server.NewFilesystemLoader(billyfs.NewAfero(afero.NewOsFs()))
	srv := server.NewServer(loader)

	ep, err := transport.NewEndpoint(gitDir)
	if err != nil {
		log.Fatal(err)
	}

	stdin := os.Stdin
	stdout := ioutil.WriteNopCloser(os.Stdout)
	// stderr := os.Stderr

	s, err := srv.NewReceivePackSession(ep, nil)
	if err != nil {
		log.Fatalf("error creating session: %s", err)
	}

	ar, err := s.AdvertisedReferences()
	if err != nil {
		log.Fatalf("internal error in advertised references: %s", err)
	}

	if err := ar.Encode(stdout); err != nil {
		log.Fatalf("error in advertised references encoding: %s", err)
	}

	req := packp.NewReferenceUpdateRequest()
	if err := req.Decode(stdin); err != nil {
		log.Fatalf("error decoding: %s", err)
	}

	rs, err := s.ReceivePack(context.TODO(), req)
	if rs != nil {
		if err := rs.Encode(stdout); err != nil {
			log.Fatalf("error in encoding report status %s", err)
		}
	}

	if err != nil {
		log.Fatalf("error in receive pack: %s", err)
	}

	return
}
