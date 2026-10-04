package config

import "maps"

type Opts int

const (
	ReturnEarly Opts = iota
)

type Config map[Opts]bool

func SetReturnEarly(val bool) Config {
	conf := make(Config)
	conf[ReturnEarly] = val
	return conf
}

func GetDefaultConfig() Config {
	return SetReturnEarly(true)
}

func MergeConfigs(configs ...Config) Config {
	mergedConfig := GetDefaultConfig()

	for _, conf := range configs {
		maps.Copy(mergedConfig, conf)
	}

	return mergedConfig
}
