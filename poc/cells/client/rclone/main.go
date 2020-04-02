package main

import (
	_ "github.com/ghecquet/tripr/poc/cells/client/rclone/backend/fs"
	"github.com/rclone/rclone/cmd"
	_ "github.com/rclone/rclone/cmd/all"    // import all commands
	_ "github.com/rclone/rclone/lib/plugin" // import plugins
)

func main() {
	cmd.Main()
}
