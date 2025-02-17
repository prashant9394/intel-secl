/*
 * Copyright (C) 2021 Intel Corporation
 * SPDX-License-Identifier: BSD-3-Clause
 */
package config

import (
	commConfig "github.com/intel-secl/intel-secl/v5/pkg/lib/common/config"
	"github.com/intel-secl/intel-secl/v5/pkg/wls/constants"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
	"os"
)

const (
	HvsBaseUrl         = "hvs-base-url"
	WlsServiceUsername = "wls.service-username"
	WlsServicePassword = "wls.service-password"
)

type Configuration struct {
	AASApiUrl        string                   `yaml:"aas-base-url" mapstructure:"aas-base-url"`
	CMSBaseURL       string                   `yaml:"cms-base-url" mapstructure:"cms-base-url"`
	CmsTlsCertDigest string                   `yaml:"cms-tls-cert-sha384" mapstructure:"cms-tls-cert-sha384"`
	HVSApiUrl        string                   `yaml:"hvs-base-url" mapstructure:"hvs-base-url"`
	WLS              commConfig.ServiceConfig `yaml:"wls"`
	TLS              commConfig.TLSCertConfig `yaml:"tls"`
	KeyCacheSeconds  int                      `yaml:"key-cache-seconds" mapstructure:"key-cache-seconds"`
	Server           commConfig.ServerConfig  `yaml:"server"`
	Log              commConfig.LogConfig     `yaml:"log"`
}

// this function sets the configure file name and type
func init() {
	viper.SetConfigName(constants.ConfigFile)
	viper.SetConfigType("yaml")
	viper.AddConfigPath(constants.ConfigDir)
}

// config is application specific
func LoadConfiguration() (*Configuration, error) {
	ret := Configuration{}
	// Find and read the config file
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// Config file not found
			return &ret, errors.Wrap(err, "Config file not found")
		}
		return &ret, errors.Wrap(err, "Failed to load config")
	}
	if err := viper.Unmarshal(&ret); err != nil {
		return &ret, errors.Wrap(err, "Failed to unmarshal config")
	}
	return &ret, nil
}

func (c *Configuration) Save(filename string) error {
	configFile, err := os.OpenFile(filename, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0600)
	if err != nil {
		return errors.Wrap(err, "Failed to create config file")
	}
	defer func() {
		derr := configFile.Close()
		if derr != nil {
			log.WithError(derr).Error("Error closing config file")
		}
	}()

	err = yaml.NewEncoder(configFile).Encode(c)
	if err != nil {
		return errors.Wrap(err, "Failed to encode config structure")
	}
	return nil
}
