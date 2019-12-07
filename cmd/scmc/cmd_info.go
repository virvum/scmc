package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"syscall"

	"github.com/virvum/scmc/pkg/mycloud"

	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh/terminal"
	"gopkg.in/yaml.v2"
)

type InfoOptions struct {
	Username string
	Password string
	Output   string
}

var infoOptions InfoOptions

var cmdInfo = &cobra.Command{
	Use:               "info [flags]",
	Short:             "Login to myCloud and show some information for the account.",
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
				return fmt.Errorf("reader.ReadString: %v\n", err)
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
				return fmt.Errorf("terminal.ReadPassword: %v\n", err)
			}

			password = string(p)
			fmt.Println()
		}

		infoOptions.Username = username
		infoOptions.Password = password

		if infoOptions.Output != "yaml" && infoOptions.Output != "json" {
			return fmt.Errorf(`invalid output format "%s"`, infoOptions.Output)
		}

		return runInfo()
	},
}

func init() {
	cmdRoot.AddCommand(cmdInfo)

	f := cmdInfo.Flags()
	f.StringVarP(&infoOptions.Username, "username", "u", os.Getenv("MYCLOUD_USERNAME"), "Swisscom myCloud username (default: $MYCLOUD_USERNAME)")
	f.StringVarP(&infoOptions.Password, "password", "p", os.Getenv("MYCLOUD_PASSWORD"), "Swisscom myCloud password (default: $MYCLOUD_PASSWORD)")
	f.StringVarP(&infoOptions.Output, "output", "o", "yaml", `output format (either "yaml" or "json")`)
}

func runInfo() error {
	mc, err := mycloud.New(infoOptions.Username, infoOptions.Password, log)
	if err != nil {
		log.Fatal("mcloud.New: %v", err)
	}

	id, err := mc.Identity()
	if err != nil {
		log.Fatal("mc.Identify: %v", err)
	}

	var s []byte

	switch infoOptions.Output {
	case "yaml":
		s, err = yaml.Marshal(id)
		if err != nil {
			log.Fatal("yaml.Marshal: %v", err)
		}
	case "json":
		s, err = json.MarshalIndent(id, "", "  ")
		if err != nil {
			log.Fatal("json.Marshal: %v", err)
		}
	}

	fmt.Println(strings.TrimSpace(string(s)))

	return nil
}
