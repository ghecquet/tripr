package main

import (
	"bufio"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"

	"github.com/fatih/color"

	"github.com/ghecquet/tripr/poc/cells/index"
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
		fs = index.NewFs(baseurl.Hostname() + "@" + baseurl.Path)
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
	case "cat":
		for _, fn := range arrCommandStr[1:] {
			var f afero.File
			if !strings.HasPrefix(fn, "/") {
				f, _ = fs.Open(cwd + "/" + fn)
			} else {
				f, _ = fs.Open(fn)
			}

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

	case "rsync":
		var err error
		var verb = strings.ToLower(arrCommandStr[1])
		if len(verb) == 0 {
			fmt.Fprintf(os.Stderr, "Error: Must provide a verb.\n")
			//printHelp()
			os.Exit(1)
		}

		fn1 := arrCommandStr[2]
		fn2 := arrCommandStr[3]

		var f1 afero.File
		if !strings.HasPrefix(fn1, "/") {
			f1, _ = fs.Open(cwd + "/" + fn1)
		} else {
			f1, _ = fs.Open(fn1)
		}

		var targetfs afero.Fs
		targeturl, _ := url.Parse(fn2)
		switch targeturl.Scheme {
		case "cells":
			targetfs = index.NewFs(targeturl.Hostname() + "@" + targeturl.Path)
		default:
			targetfs = afero.NewBasePathFs(afero.NewOsFs(), targeturl.Path)
		}

		f2, err := targetfs.Open("/")

		switch verb {
		case "signature":
			err = signature(f1, os.Stdout, 1024)
		// case "delta":
		// 	err = delta(fl.Arg(1), fl.Arg(2), fl.Arg(3), *checkFile, deltaComp)
		// case "patch":
		// 	err = patch(f1, f2,  false)
		case "test":

			err = test(f1, f2)

			fmt.Println(err)
		default:
			fmt.Fprintf(os.Stderr, "Error: Unrecognized verb: %s\n", verb)
			// printHelp()
			os.Exit(1)
		}

		if err != nil {
			fmt.Fprintf(os.Stderr, "Error in %s: %s", verb, err)
			os.Exit(2)
		}

	case "exit":
		os.Exit(0)
		// add another case here for custom commands.
	}

	return nil
}
