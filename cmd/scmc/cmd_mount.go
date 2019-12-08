package main

import (
	"os"

	"github.com/spf13/cobra"
)

// MountOptions represents options for the command "mount".
type MountOptions struct {
	Username string
	Password string
}

var mountOptions MountOptions

var cmdMount = &cobra.Command{
	Use:               "mount [flags] mountpoint",
	Short:             "mount myCloud drive locally (NOT YET IMPLEMENTED).",
	DisableAutoGenTag: true,
	Args:              cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runMount(args[0])
	},
}

func init() {
	cmdRoot.AddCommand(cmdMount)

	f := cmdMount.Flags()
	f.StringVarP(&mountOptions.Username, "username", "u", os.Getenv("MYCLOUD_USERNAME"), "Swisscom myCloud username")
	f.StringVarP(&mountOptions.Password, "password", "p", os.Getenv("MYCLOUD_PASSWORD"), "Swisscom myCloud password")
}

func runMount(mountpoint string) error {
	// TODO
	// TODO umount: systemFuse.Unmount(mountpoint)

	return nil
}
