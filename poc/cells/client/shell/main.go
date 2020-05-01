package main

import (
	"bufio"
	"bytes"
	"compress/zlib"
	"crypto/sha1"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"

	"github.com/fatih/color"
	"github.com/ghecquet/tripr/poc/cells/aferofs"
	"github.com/ghecquet/tripr/poc/cells/billyfs"
	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/config"
	"gopkg.in/src-d/go-git.v4/plumbing/cache"
	"gopkg.in/src-d/go-git.v4/plumbing/transport/client"
	"gopkg.in/src-d/go-git.v4/plumbing/transport/file"
	"gopkg.in/src-d/go-git.v4/storage/filesystem"
	"gopkg.in/src-d/go-git.v4/storage/memory"

	"github.com/minio/sha256-simd"

	"github.com/spf13/afero"
)

var (
	fs  afero.Fs
	cwd string
)

func init() {
}

func main() {
	reader := bufio.NewReader(os.Stdin)

	cwd = "/"

	basefs := os.Args[1]
	baseurl, _ := url.Parse(basefs)

	switch baseurl.Scheme {
	case "cells":
		fs = aferofs.NewIndexFs(baseurl.Hostname() + "@" + baseurl.Path)
	default:
		fs = afero.NewBasePathFs(afero.NewOsFs(), baseurl.Path)
	}

	for {
		color.Set(color.FgCyan, color.Bold)
		fmt.Printf("[ %s ] $ ", cwd)
		color.Unset()

		cmdString, err := reader.ReadString('\n')
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
		err = runCommand(cmdString)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
	}
}

func runCommand(commandStr string) error {
	commandStr = strings.TrimSuffix(commandStr, "\n")
	arrCommandStr := strings.Fields(commandStr)
	switch arrCommandStr[0] {
	case "cd":
		if len(arrCommandStr) == 1 {
			cwd = "/"
		} else {
			cwd = filepath.Clean(cwd + "/" + arrCommandStr[1])
		}

	case "ls":
		cwdf, err := fs.Open(cwd)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		defer cwdf.Close()

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', tabwriter.TabIndent)

		files, _ := cwdf.Readdir(-1)
		for _, file := range files {
			fmt.Fprintf(w, "%s\t%s\n", os.FileMode(file.Mode()), file.Name())
		}
		w.Flush()
	case "mkdir":
		fn := filepath.Clean(cwd + "/" + arrCommandStr[1])

		if err := fs.Mkdir(fn, os.ModePerm); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	case "cat":
		for _, fn := range arrCommandStr[1:] {
			var f afero.File
			if !strings.HasPrefix(fn, "/") {
				f, _ = fs.Open(cwd + "/" + fn)
			} else {
				f, _ = fs.Open(fn)
			}
			defer f.Close()

			buf := make([]byte, 1024)
			for {
				n, err := io.CopyBuffer(os.Stdout, f, buf)
				if err != nil && err != io.EOF {
					break
				}

				if n == 0 {
					break
				}
			}
			fmt.Println()
		}
	case "write":
		fn := arrCommandStr[1]
		var f afero.File
		var err error
		if !strings.HasPrefix(fn, "/") {
			f, err = fs.Create(cwd + "/" + fn)
		} else {
			f, err = fs.Create(fn)
		}

		if err != nil {
			fmt.Println(err)
			os.Exit(0)
		}
		defer f.Close()

		f.WriteString(arrCommandStr[2])

	case "clone":

		osfs := afero.NewOsFs()
		gitfs := afero.NewBasePathFs(osfs, "/tmp/test1git/.git")

		type blob struct {
			oid  []byte
			size int64
		}

		const numIndexBlobsWorkers = 100
		blobs := make(chan blob)

		for i := 0; i < numIndexBlobsWorkers; i++ {
			go func() {
				for b := range blobs {
					fmt.Printf("%x %d\n", b.oid, b.size)
				}
			}()
		}

		go func() {

		}()

		worktreefs := afero.NewBasePathFs(fs, cwd)
		gitstorage := filesystem.NewStorage(billyfs.NewAfero(gitfs), cache.NewObjectLRUDefault())

		_, err := git.Init(gitstorage, billyfs.NewAfero(worktreefs))
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		// idx, err := gitstorage.Index()
		// if err != nil {
		// 	log.Fatal(err)
		// }

		// filehash := [20]byte{}
		// var entries []*index.Entry

		max := make(chan struct{}, 16)

		afero.Walk(worktreefs, "/", func(path string, fi os.FileInfo, err error) error {
			if fi.IsDir() {
				return nil
			}

			max <- struct{}{}
			go func(p string) {
				file, err := worktreefs.Open(p)
				if err != nil {
					fmt.Println(err)
					<-max
					return
				}
				defer func(f afero.File) {
					file.Close()
					<-max
				}(file)

				// Git LFS ref file needs to be encoded in sha256
				sha256Writer := sha256.New()
				pathlen := len(path)
				pathlenb := make([]byte, 4)
				binary.LittleEndian.PutUint32(pathlenb, uint32(pathlen))

				sha256Writer.Write([]byte("file "))
				sha256Writer.Write(pathlenb)
				sha256Writer.Write([]byte("\\0"))
				sha256Writer.Write([]byte(path))

				buf := new(bytes.Buffer)
				fmt.Fprintf(buf, "version https://git-lfs.github.com/spec/v1\n")
				// fmt.Fprintf(buf, "oid sha256:%x\n", sha256Writer.Sum(nil))
				// fmt.Fprintf(buf, "size %d\n", pathlen)

				fmt.Fprintf(buf, "oid sha256:c98c24b677eff44860afea6f493bbaec5bb1c4cbb209c6fc2bbb47f66ff2ad31\n")
				fmt.Fprintf(buf, "size 14\n")

				sha1Writer := sha1.New()
				buflen := buf.Len()
				buflenb := make([]byte, 4)
				binary.LittleEndian.PutUint32(buflenb, uint32(buflen))
				sha1Writer.Write([]byte("blob "))
				sha1Writer.Write(buflenb)
				sha1Writer.Write([]byte("\\0"))
				sha1Writer.Write(buf.Bytes())
				sig := fmt.Sprintf("%x", sha1Writer.Sum(nil))

				gitfs.MkdirAll(filepath.Join("objects", sig[0:2]), 0700)
				fd, err := gitfs.Create(filepath.Join("objects", sig[0:2], sig[2:]))
				if err != nil {
					fmt.Println("Could not open ", err)
					return
				}
				defer fd.Close()

				zwriter := zlib.NewWriter(fd)
				defer zwriter.Close()

				writ, err := zwriter.Write(buf.Bytes())
				fmt.Println(writ, err)
			}(path)

			// if !fi.IsDir() {
			// 	mode, _ := filemode.NewFromOSFileMode(fi.Mode())
			// 	entries = append(entries, &index.Entry{
			// 		Name: path,
			// 		Hash: filehash,
			// 		Mode: mode,
			// 	})
			// }

			return nil
		})

		// idx.Entries = entries
		// gitstorage.SetIndex(idx)

		// worktree, _ := r.Worktree()

		// fmt.Println("Starting commit")

		// hash, _ := worktree.Commit("Initial version", &git.CommitOptions{Author: &object.Signature{
		// 	Name:  "John Doe",
		// 	Email: "john@doe.org",
		// 	When:  time.Now(),
		// }})

		// fmt.Println(hash)

		// idx.Add()

		// // m := memory.NewStorage()
		// r, err := git.Init(m, billyfs.NewAfero(afero.NewBasePathFs(fs, cwd)))
		// if err != nil {
		// 	fmt.Println(err)
		// 	os.Exit(1)
		// }

		// wt, err := r.Worktree()

		// if _, err := wt.Add("."); err != nil {
		// 	fmt.Println(err)
		// 	return err
		// }

		// status, err := wt.Status()
		// if err != nil {
		// 	return err
		// }

		// fmt.Println(status)

		// hash, err := wt.Commit("Initial version", &git.CommitOptions{Author: &object.Signature{
		// 	Name:  "John Doe",
		// 	Email: "john@doe.org",
		// 	When:  time.Now(),
		// }})
		// if err != nil {
		// 	fmt.Println(err)
		// 	os.Exit(1)
		// }

		// fmt.Println(hash)

		// targetStorage := memory.NewStorage()
		// targetFs := billyfs.NewAfero(afero.NewBasePathFs(osfs, "/tmp/test4"))

		// _, err := git.Clone(targetStorage, targetFs, &git.CloneOptions{
		// 	URL: "/tmp/test1",
		// })
		// if err != nil {
		// 	log.Fatal(err)
		// }

		// if err := r.Clone(memory.NewStorage(), git2.New(afero.NewBasePathFs(afero.NewOsFs(), "/tmp/test3"))); err != nil {
		// 	fmt.Println(err)
		// 	os.Exit(1)
		// }
	case "test":
		osfs := afero.NewOsFs()
		r, err := git.Init(memory.NewStorage(), billyfs.NewAfero(afero.NewBasePathFs(osfs, "/tmp/test3")))
		if err != nil {
			log.Fatal(err)
		}

		client.InstallProtocol("cells", file.NewClient("cells-upload-pack", "cells-receive-pack"))

		// Add a new remote, with the default fetch refspec
		_, err = r.CreateRemote(&config.RemoteConfig{
			Name: "example",
			URLs: []string{"cells:///tmp/test1"},
		})

		if err != nil {
			log.Fatal(err)
		}

		// List remotes from a repository
		list, err := r.Remotes()
		if err != nil {
			log.Fatal(err)
		}

		for _, r := range list {
			fmt.Println(r)
		}

		w, _ := r.Worktree()

		if err := w.Pull(&git.PullOptions{
			RemoteName: "example",
			Force:      true,
		}); err != nil {
			log.Fatal("HEEERRRE ", err)
		}

	case "exit":
		os.Exit(0)
		// add another case here for custom commands.
	}

	return nil
}
