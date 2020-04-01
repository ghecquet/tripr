package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/ghecquet/tripr/poc/cells/index"
	"github.com/spf13/afero"
)

var (
	fs  afero.Fs
	cwd string
)

func init() {
	fs = index.NewFs()
	cwd = "/"
}

func main() {
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("$ ")
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
		cwdf, _ := fs.Open(cwd)
		defer cwdf.Close()

		files, _ := cwdf.Readdir(-1)
		for _, file := range files {
			fmt.Println(file.Name())
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

			data, err := ioutil.ReadAll(f)
			if err != nil {
				return err
			}
			fmt.Println(string(data))
		}
	case "exit":
		os.Exit(0)
		// add another case here for custom commands.
	}

	return nil
}
