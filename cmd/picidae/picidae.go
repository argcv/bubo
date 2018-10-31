package main

import (
	"fmt"
	"github.com/argcv/picidae/assets"
	"github.com/argcv/picidae/cmd/picidae/command"
	"github.com/argcv/picidae/pkg/utils"
	"github.com/argcv/picidae/version"
	"github.com/argcv/stork/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"math/rand"
	"os"
	"path"
	"time"
)

var (
	binCleanName = path.Clean(os.Args[0])
	versionMsg   = fmt.Sprintf("%v version \"%s (%s)\" %s\n", binCleanName, version.Version, version.GitHash, version.BuildDate)
	rootCmd      = &cobra.Command{
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if verbose, err := cmd.Flags().GetBool("verbose"); err == nil {
				if verbose {
					log.Verbose()
					log.Debug(versionMsg)
					log.Debug("verbose mode: ON")
				}
			}

			// init config
			conf, _ := cmd.Flags().GetString("config")
			if e := assets.LoadConfig(conf); e != nil {
				return e
			}

			// init system variables
			rand.Seed(time.Now().Unix())
			utils.SetMaxProcs()

			return nil
		},
	}
	versionCmd = &cobra.Command{
		Use:   "version",
		Short: fmt.Sprintf("Prints the version of %s", binCleanName),
		// do not execute any persistent actions
		PersistentPreRun: func(cmd *cobra.Command, args []string) {},
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Print(versionMsg)
		},
	}
)

func init() {
	rootCmd.AddCommand(
		versionCmd,
		command.NewClientCommand(),
		command.NewServerCommand(),
		command.NewToolCommand(),
	)
	rootCmd.PersistentFlags().BoolP("debug", "d", false, "debug mode")
	rootCmd.PersistentFlags().StringP("config", "c", "", "explicit assign a configuration file")
	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "log verbose")

	viper.BindPFlag("debug", rootCmd.PersistentFlags().Lookup("debug"))
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		log.Errorf("Execute Failed!!! %v", err.Error())
		os.Exit(1)
	}
}
