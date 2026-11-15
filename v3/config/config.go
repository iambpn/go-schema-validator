// Package config holds the options for struct validation in go-schema-validator.
package config

import "maps"

// Opts names one validation option. These are the options:
//
//   - ReturnEarly: if true, struct validation stops at the first failed field and returns one error.
//     If false, it checks every field and returns one error for each failed field. The default is true.
//     Set it with SetReturnEarly.
type Opts int

const (
	// ReturnEarly controls whether struct validation stops at the first failed field. The default is true.
	ReturnEarly Opts = iota
)

// Config maps options to their values. An option that is not in the map keeps its default value.
type Config map[Opts]bool

// SetReturnEarly returns a Config that sets the ReturnEarly option to val.
func SetReturnEarly(val bool) Config {
	conf := make(Config)
	conf[ReturnEarly] = val
	return conf
}

// GetDefaultConfig returns a new Config with the default value of every option.
func GetDefaultConfig() Config {
	return SetReturnEarly(true)
}

// MergeConfigs returns the default Config with each given Config applied in order.
// If two configs set the same option, the later config wins.
func MergeConfigs(configs ...Config) Config {
	mergedConfig := GetDefaultConfig()

	for _, conf := range configs {
		maps.Copy(mergedConfig, conf)
	}

	return mergedConfig
}
