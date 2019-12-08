package main

import (
	"bufio"
	"fmt"
	"os"
	"syscall"

	"github.com/virvum/scmc/pkg/mycloud"

	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh/terminal"
)

// CheckOptions represents options for the command "check".
type CheckOptions struct {
	Username string
	Password string
}

var checkOptions CheckOptions

var cmdCheck = &cobra.Command{
	Use:               "check [flags]",
	Short:             "Login to myCloud and and test all reverse-engineered myCloud API calls.",
	DisableAutoGenTag: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		var (
			username string
			password string
		)

		// TODO move this shit to aux.go or somewhere else

		reader := bufio.NewReader(os.Stdin)

		if cmd.Flags().Changed("username") {
			username = cliOptions.Username
		} else if cfg.Username != "" {
			username = cfg.Username
		} else {
			fmt.Print("Swisscom myCloud username: ")
			u, err := reader.ReadString('\n')
			if err != nil {
				return fmt.Errorf("reader.ReadString: %v", err)
			}

			username = u
		}

		if cmd.Flags().Changed("password") {
			password = cliOptions.Password
		} else if cfg.Password != "" {
			password = cfg.Password
		} else {
			fmt.Print("Swisscom myCloud password: ")
			p, err := terminal.ReadPassword(int(syscall.Stdin))
			if err != nil {
				return fmt.Errorf("terminal.ReadPassword: %v", err)
			}

			password = string(p)
			fmt.Println()
		}

		checkOptions.Username = username
		checkOptions.Password = password

		return runCheck()
	},
}

func init() {
	cmdRoot.AddCommand(cmdCheck)

	// TODO omit cobra default value if $MYCLOUD_* is set
	f := cmdCheck.Flags()
	f.StringVarP(&checkOptions.Username, "username", "u", os.Getenv("MYCLOUD_USERNAME"), "Swisscom myCloud username (default: $MYCLOUD_USERNAME)")
	f.StringVarP(&checkOptions.Password, "password", "p", os.Getenv("MYCLOUD_PASSWORD"), "Swisscom myCloud password (default: $MYCLOUD_PASSWORD)")
}

// Check represents a single check name and its function containing the check's logic.
type Check struct {
	Name string
	Fn   func(*mycloud.MyCloud) error
}

var checks []Check = []Check{
	{
		Name: "authentication",
		Fn: func(mc *mycloud.MyCloud) error {
			c, err := mycloud.New(checkOptions.Username, checkOptions.Password, log)
			if err != nil {
				return fmt.Errorf("mcloud.New: %v", err)
			}

			mc = c

			return nil
		},
	},
	{
		Name: "fetch identity",
		Fn: func(mc *mycloud.MyCloud) error {
			_, err := mc.Identity()
			if err != nil {
				return fmt.Errorf("mc.Identity: %v", err)
			}

			return nil
		},
	},
	{
		Name: "fetch usage",
		Fn: func(mc *mycloud.MyCloud) error {
			_, err := mc.Usage()
			if err != nil {
				return fmt.Errorf("mc.Usage: %v", err)
			}

			return nil
		},
	},
}

func runCheck() error {
	var mc mycloud.MyCloud

	total := len(checks)

	for i, check := range checks {
		err := check.Fn(&mc)
		if err != nil {
			fmt.Printf("[%2d/%d] %s: %v\n\nA check failed.\n", i+1, total, check.Name, err)
			return nil
		}

		fmt.Printf("[%2d/%d] %s: OK\n", i+1, total, check.Name)
		// TODO flush stdout
	}

	fmt.Println("All checks successfully executed.")

	return nil
}
