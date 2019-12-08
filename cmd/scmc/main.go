package main

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"

	"github.com/virvum/scmc/internal/config"
	"github.com/virvum/scmc/pkg/logger"

	"github.com/spf13/cobra"
)

// GlobalOptions represents global options for all commands.
type GlobalOptions struct {
	ConfigFile string
	LogLevel   logger.Level
}

var (
	globalOptions GlobalOptions
	cfg           config.Config
	log           logger.Log
	rootPath      string
	buildVersion  string = "?"
	buildDate     string = "?"
)

var cmdRoot = &cobra.Command{
	Use:   "scmc",
	Short: "Swisscom myCloud client",
	Long:  "scmc is a program to interact with the Swisscom myCloud service in a number of different ways.",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		var (
			err error
			c   *config.Config = nil
		)

		if cmd.Flags().Changed("log-level") {
			log.Level = globalOptions.LogLevel
		}

		if cmd.Flags().Changed("config-file") {
			log.Debug("config-file set, trying %s", globalOptions.ConfigFile)

			c, err = config.Load(globalOptions.ConfigFile, &log)
			if err != nil {
				return err
			}
		} else {
			log.Debug("config-file not set, trying default paths")

			usr, err := user.Current()
			if err == nil {
				fn := filepath.Join(usr.HomeDir, ".scmc.yaml")

				log.Debug("trying %s", fn)

				if _, err := os.Stat(fn); err == nil {
					c, err = config.Load(fn, &log)
					if err != nil {
						return err
					}
				} else {
					log.Debug("%s not found", fn)
				}
			} else {
				log.Debug("unable to get user home")
			}

			if c == nil {
				fn := "/etc/scmc.yaml"

				log.Debug("trying %s", fn)

				if _, err := os.Stat(fn); err == nil {
					c, err = config.Load(fn, &log)
					if err != nil {
						return err
					}
				} else {
					log.Debug("%s not found", fn)
				}
			}
		}

		if c != nil {
			cfg = *c
		}

		if cmd.Flags().Changed("log-level") {
			cfg.LogLevel = globalOptions.LogLevel
		}

		log.Level = cfg.LogLevel

		log.Debug("loaded configuration: %+v", cfg)

		return nil
	},
}

func init() {
	globalOptions.LogLevel = logger.Warn

	f := cmdRoot.PersistentFlags()
	f.StringVarP(&globalOptions.ConfigFile, "config-file", "c", "", `path to configuration file (if not specified, "$HOME/.scmc.yaml" is tried first, then "/etc/scmc.yaml")`)
	f.VarP(&globalOptions.LogLevel, "log-level", "l", fmt.Sprintf("log level (either %s)", oxfordJoin(logger.LogLevels, `"%s"`, "or")))
}

func main() {
	// Default values (must also be set in `internal/config/main.go:Load()`).
	cfg.LogLevel = logger.Warn

	log = logger.New(cfg.LogLevel, true, rootPath)

	if err := cmdRoot.Execute(); err != nil {
		log.Fatal("cmdRoot.Execute: %v", err)
	}
}
