package assets

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
)

type PicidaeSrvConfig struct {
	Bind string `mapstructure:"bind"`
	Port int    `mapstructure:"port"`

	Services []struct {
		Secret string `mapstructure:"secret"`
		Port   int    `mapstructure:"port"`
	} `mapstructure:"services"`
}

type PicidaeCliConfig struct {
	Host string `mapstructure:"host"`
	Port int    `mapstructure:"port"`

	Services []struct {
		Local struct {
			Host string `mapstructure:"host"`
			Port int    `mapstructure:"port"`
		} `mapstructure:"local"`
		Remote struct {
			Port int `mapstructure:"port"`
		} `mapstructure:"remote"`
		Secret string `mapstructure:"secret"`
	}
}

type PicidaeConfig struct {
	Debug  bool              `mapstructure:"debug"`
	Client *PicidaeCliConfig `mapstructure:"client"`
	Server *PicidaeSrvConfig `mapstructure:"server"`
}

func LoadPicidaeSrvConfig() (cfg *PicidaeSrvConfig, err error) {
	sub := viper.GetViper().Sub(KeyServer)
	if sub == nil {
		return nil, errors.New(fmt.Sprintf("key[%s] not found", KeyServer))
	}

	cfg = &PicidaeSrvConfig{}
	err = sub.Unmarshal(&cfg)
	return
}

func LoadPicidaeCliConfig() (cfg *PicidaeCliConfig, err error) {
	sub := viper.GetViper().Sub(KeyClient)
	if sub == nil {
		return nil, errors.New(fmt.Sprintf("key[%s] not found", KeyClient))
	}

	cfg = &PicidaeCliConfig{}
	err = sub.Unmarshal(&cfg)
	return
}

func LoadPicidaeConfig() (cfg *PicidaeConfig, err error) {
	cfg = &PicidaeConfig{}
	err = viper.Unmarshal(&cfg)
	return
}
