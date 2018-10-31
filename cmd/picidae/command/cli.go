package command

import (
	"context"
	"errors"
	"fmt"
	"github.com/argcv/picidae/assets"
	"github.com/argcv/picidae/pkg/cli"
	"github.com/argcv/stork/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"os"
	"os/signal"
	"time"
)

func NewClientCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use: "cli",
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			cfg, e := assets.LoadPicidaeConfig()
			if e != nil {
				return e
			}

			cliCfg := cfg.Client

			if cliCfg == nil {
				return errors.New("server config is empty")
			}

			debug := cfg.Debug
			if debug {
				log.Verbose()
				log.Infof("Mode: Debug")
			} else {
				log.Infof("Mode: Release")
			}

			log.Infof("Requesting: %v:%v", cliCfg.Host, cliCfg.Port)

			s, e := cli.NewManager(cliCfg)
			if e != nil {
				msg := fmt.Sprintf("Init Manager failed: %v", e)
				log.Errorf(msg)
				return errors.New(msg)
			}

			go func() {
				s.Start()
			}()

			quit := make(chan os.Signal)
			signal.Notify(quit, os.Interrupt)
			<-quit
			log.Infof("Shutdown Server ...")

			signal.Reset(os.Interrupt)

			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			log.Infof("Server shutting down")
			if err := s.Shutdown(ctx); err != nil {
				log.Fatal("Server Shutdown:", err)
			}
			log.Infof("Server exiting")
			return err
		},
	}
	cmd.PersistentFlags().String("host", "127.0.0.1", "Http Server Host")
	cmd.PersistentFlags().IntP("port", "p", 63123, "Http Server Port")

	viper.BindPFlag(assets.KeyRPCSrvBind, cmd.PersistentFlags().Lookup("host"))
	viper.BindPFlag(assets.KeyRPCSrvPort, cmd.PersistentFlags().Lookup("port"))
	return cmd
}
