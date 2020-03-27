package index

import (
	context "context"
	"fmt"
	"os"

	"github.com/spf13/afero"
)

type Handler struct {
	fs afero.Fs
}

func NewHandler(fs afero.Fs) *Handler {
	return &Handler{
		fs: fs,
	}
}

func (h *Handler) ReadNode(ctx context.Context, in *ReadNodeRequest) (*ReadNodeResponse, error) {
	return nil, fmt.Errorf("Not implemented")
}

func (h *Handler) ListNodes(in *ListNodesRequest, stream NodeProvider_ListNodesServer) error {
	afero.Walk(h.fs, "/", func(path string, fi os.FileInfo, err error) error {
		return stream.Send(&ListNodesResponse{
			Node: &Node{
				Path: path,
			},
		})
	})

	return fmt.Errorf("Not implemented")
}
