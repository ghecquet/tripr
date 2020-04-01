package main

import (
	"fmt"
	"os"
	"time"

	_ "github.com/ghecquet/tripr/poc/cells/client/resolver"
	"github.com/spf13/afero"
)

func main() {

	fs1 := afero.NewBasePathFs(afero.NewOsFs(), "/tmp/test1")
	fs2 := afero.NewBasePathFs(afero.NewOsFs(), "/tmp/test2")

	fmt.Println("=================================")
	fmt.Println("-- FS1")
	fmt.Println("=================================")
	afero.Walk(fs1, "/", func(path string, fi os.FileInfo, err error) error {
		fmt.Println(path)

		return nil
	})

	fmt.Println("=================================")
	fmt.Println("-- FS2")
	fmt.Println("=================================")
	afero.Walk(fs2, "/", func(path string, fi os.FileInfo, err error) error {
		fmt.Println(path)

		return nil
	})

	fmt.Println("=================================")
	fmt.Println("-- Composite")
	fmt.Println("=================================")
	fs := afero.NewCacheOnReadFs(fs1, fs2, 10*time.Second)
	// fs := compositefs.NewCompositeFs(fs1, fs2)
	afero.Walk(fs, "/", func(path string, fi os.FileInfo, err error) error {
		fmt.Println(path)

		return nil
	})

	// fs := index.NewIndexFs()

	// afero.Walk(fs, "/tmp", func(path string, fi os.FileInfo, err error) error {
	// 	fmt.Println(path)

	// 	return nil
	// })
}
