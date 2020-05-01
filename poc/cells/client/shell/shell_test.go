package main

import (
	"fmt"
	"io"
	"os"
	"testing"

	"github.com/ghecquet/tripr/poc/cells/aferofs"
	"github.com/ghecquet/tripr/poc/cells/fastwalk"
	"github.com/minio/sha256-simd"
	"github.com/spf13/afero"
)

var stdout io.Writer
var worktreefs afero.Fs

func init() {
	stdout = os.NewFile(0, os.DevNull)
	//stdout = os.Stdout
	fs = aferofs.NewFastReadFs(afero.NewOsFs())
	// cwd = "/"
	// worktreefs = afero.NewBasePathFs(fs, cwd)
}

func BenchmarkFastWalk(b *testing.B) {
	max := make(chan struct{}, 16)

	fastwalk.Walk(afero.NewOsFs(), "/Users/ghecquet/Documents", func(path string, mode os.FileMode) error {
		if mode.IsDir() {
			return nil
		}

		max <- struct{}{}
		go func(p string) {
			file, err := fs.Open(p)
			if err != nil {
				fmt.Println(err)
				<-max
				return
			}
			defer func(f afero.File) {
				file.Close()
				<-max
			}(file)

			shaWriter := sha256.New()
			written, _ := io.Copy(shaWriter, file)

			fmt.Fprintf(stdout, "%x %d\n", shaWriter.Sum(nil), written)
		}(path)

		return nil
	})
}

func BenchmarkSimple(b *testing.B) {
	max := make(chan struct{}, 16)

	afero.Walk(afero.NewOsFs(), "/Users/ghecquet/Documents", func(path string, fi os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		if fi.IsDir() {
			return nil
		}

		max <- struct{}{}
		go func(p string) {
			file, err := fs.Open(p)
			if err != nil {
				fmt.Println(err)
				<-max
				return
			}
			defer func(f afero.File) {
				file.Close()
				<-max
			}(file)

			shaWriter := sha256.New()
			written, _ := io.Copy(shaWriter, file)

			fmt.Fprintf(stdout, "%x %d\n", shaWriter.Sum(nil), written)
		}(path)

		return nil
	})
}

// func BenchmarkSimpleWithBlobWorkers(b *testing.B) {
// 	type blob struct {
// 		oid  []byte
// 		size int64
// 	}

// 	const numIndexBlobsWorkers = 100
// 	blobs := make(chan blob)

// 	for i := 0; i < numIndexBlobsWorkers; i++ {
// 		go func() {
// 			for b := range blobs {
// 				fmt.Fprintf(stdout, "%x %d\n", b.oid, b.size)
// 			}
// 		}()
// 	}

// 	b.ResetTimer()

// 	max := make(chan struct{}, 16)

// 	afero.Walk(worktreefs, "/", func(path string, fi os.FileInfo, err error) error {
// 		if err != nil {
// 			return nil
// 		}

// 		if fi.IsDir() {
// 			return nil
// 		}

// 		max <- struct{}{}
// 		go func(p string) {
// 			file, err := worktreefs.Open(p)
// 			if err != nil {
// 				fmt.Println(err)
// 				<-max
// 				return
// 			}
// 			defer func(f afero.File) {
// 				file.Close()
// 				<-max
// 			}(file)

// 			shaWriter := sha256.New()
// 			written, _ := io.Copy(shaWriter, file)

// 			blobs <- blob{
// 				oid:  shaWriter.Sum(nil),
// 				size: written,
// 			}
// 		}(path)

// 		return nil
// 	})
// }

// func BenchmarkFastWalkWithBlobWorkers(b *testing.B) {
// 	type blob struct {
// 		oid  []byte
// 		size int64
// 	}

// 	const numIndexBlobsWorkers = 100
// 	blobs := make(chan blob)

// 	for i := 0; i < numIndexBlobsWorkers; i++ {
// 		go func() {
// 			for b := range blobs {
// 				fmt.Fprintf(stdout, "%x %d\n", b.oid, b.size)
// 			}
// 		}()
// 	}

// 	b.ResetTimer()

// 	max := make(chan struct{}, 16)

// 	fastwalk.Walk(worktreefs, "/", func(path string, mode os.FileMode) error {
// 		if mode.IsDir() {
// 			return nil
// 		}

// 		max <- struct{}{}
// 		go func(p string) {
// 			file, err := worktreefs.Open(p)
// 			if err != nil {
// 				fmt.Println(err)
// 				<-max
// 				return
// 			}
// 			defer func(f afero.File) {
// 				file.Close()
// 				<-max
// 			}(file)

// 			shaWriter := sha256.New()
// 			written, _ := io.Copy(shaWriter, file)

// 			blobs <- blob{
// 				oid:  shaWriter.Sum(nil),
// 				size: written,
// 			}
// 		}(path)

// 		return nil
// 	})
// }
