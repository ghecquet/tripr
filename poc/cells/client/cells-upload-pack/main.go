package main

import (
	"context"
	"log"
	"os"

	"github.com/ghecquet/tripr/poc/cells/aferofs"
	"github.com/ghecquet/tripr/poc/cells/billyfs"
	"github.com/spf13/afero"
	"gopkg.in/src-d/go-git.v4/plumbing/protocol/packp"
	"gopkg.in/src-d/go-git.v4/plumbing/transport"
	"gopkg.in/src-d/go-git.v4/plumbing/transport/server"
	"gopkg.in/src-d/go-git.v4/utils/ioutil"
)

func main() {
	// Loading filesystem directly
	worktree := afero.NewBasePathFs(afero.NewOsFs(), "/tmp/test1")
	gitdir := afero.NewBasePathFs(afero.NewOsFs(), "/tmp/test1git")
	loader := server.NewFilesystemLoader(billyfs.NewAfero(aferofs.NewGitDirFs(worktree, gitdir)))
	srv := server.NewServer(loader)

	ep, err := transport.NewEndpoint("/.git")
	if err != nil {
		log.Fatal(err)
	}

	stdin := os.Stdin
	stdout := ioutil.WriteNopCloser(os.Stdout)

	// TODO: define and implement a server-side AuthMethod
	s, err := srv.NewUploadPackSession(ep, nil)
	if err != nil {
		log.Fatalf("error creating session: %s", err)
	}

	// ioutil.CheckClose(stdout, &err)

	ar, err := s.AdvertisedReferences()
	if err != nil {
		log.Fatal(err)
	}

	if err := ar.Encode(stdout); err != nil {
		log.Fatal(err)
	}

	req := packp.NewUploadPackRequest()
	if err := req.Decode(stdin); err != nil {
		log.Fatal(err)
	}

	var resp *packp.UploadPackResponse
	resp, err = s.UploadPack(context.TODO(), req)
	if err != nil {
		log.Fatal(err)
	}

	if err := resp.Encode(stdout); err != nil {
		log.Fatal(err)
	}

	return
}
