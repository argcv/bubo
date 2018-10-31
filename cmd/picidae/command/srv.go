package command

import (
	"context"
	"fmt"
	"github.com/argcv/picidae/assets"
	"github.com/argcv/picidae/pkg/srv"
	"github.com/argcv/stork/log"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"os"
	"os/signal"
	"time"
)

func NewServerCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use: "srv",
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			cfg, e := assets.LoadPicidaeConfig()
			if e != nil {
				return e
			}

			srvCfg := cfg.Server

			if srvCfg == nil {
				return errors.New("server config is empty")
			}

			debug := cfg.Debug
			if debug {
				log.Verbose()
				log.Infof("Mode: Debug")
			} else {
				log.Infof("Mode: Release")
			}

			// continue...
			//httpHost := config.GetHttpHost()
			rpcBind := srvCfg.Bind
			rpcPort := srvCfg.Port

			listenStr := fmt.Sprintf("%v:%v", rpcBind, rpcPort)

			log.Infof("Listening: [%v]", listenStr)

			s, e := srv.NewManager(srvCfg)
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

			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			log.Infof("Server shutting down")
			if err := s.Shutdown(ctx); err != nil {
				log.Fatal("Server Shutdown:", err)
			}
			log.Infof("Server exiting")
			//r.Run(listenStr)
			return
		},
	}
	//cmd.PersistentFlags().BoolP("debug", "d", false, "debug mode")
	cmd.PersistentFlags().String("host", "0.0.0.0", "Http Server Host")
	cmd.PersistentFlags().IntP("port", "p", 63123, "Http Server Port")

	//viper.BindPFlag("debug", cmd.PersistentFlags().Lookup("debug"))
	viper.BindPFlag(assets.KeyRPCSrvBind, cmd.PersistentFlags().Lookup("host"))
	viper.BindPFlag(assets.KeyRPCSrvPort, cmd.PersistentFlags().Lookup("port"))
	return cmd
}
