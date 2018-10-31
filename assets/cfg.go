package assets

import (
	"fmt"
	"github.com/argcv/stork/config"
	"os"
)

type ConfigInfo struct {
	Error error
}

var (
	GlobalConfig ConfigInfo = ConfigInfo{}
)

func init() {
	// anything to init?
}

func LoadConfig(path string) (err error) {
	options := []config.Option{
		{
			Project:        ProjectName,
			FileMustExists: false,
		},
	}

	if path != "" {
		options = append(options, config.Option{
			Path: path,
		})
	}

	if gopath := os.Getenv("GOPATH"); len(gopath) > 0 {
		options = append(options, config.Option{
			ConfigFallbackSearchPath: fmt.Sprintf("$GOPATH/src/github.com/argcv/%s/", ProjectName),
		})
	}


	err = config.LoadConfig(options...)
	GlobalConfig.Error = err
	return
}
