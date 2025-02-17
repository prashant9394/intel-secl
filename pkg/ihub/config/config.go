/*
 * Copyright (C) 2022 Intel Corporation
 * SPDX-License-Identifier: BSD-3-Clause
 */
package config

import (
	"github.com/intel-secl/intel-secl/v5/pkg/ihub/constants"
	log "github.com/sirupsen/logrus"
	"os"

	commConfig "github.com/intel-secl/intel-secl/v5/pkg/lib/common/config"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

const (
	PollIntervalMinutes = "poll-interval-minutes"
	IhubServiceUsername = "ihub.service-username"
	IhubServicePassword = "ihub.service-password"
	HvsBaseUrl          = "attestation-service.hvs-base-url"
	ShvsBaseUrl         = "attestation-service.shvs-base-url"
	Tenant              = "tenant"
	KubernetesUrl       = "kubernetes-url"
	KubernetesCrd       = "kubernetes-crd"
	KubernetesToken     = "kubernetes-token"
	KubernetesCertFile  = "kubernetes-cert-file"
)

// Configuration is the global configuration struct that is marshalled/unmarshalled to a persisted yaml file
type Configuration struct {
	AASBaseUrl          string `yaml:"aas-base-url" mapstructure:"aas-base-url"`
	CMSBaseURL          string `yaml:"cms-base-url" mapstructure:"cms-base-url"`
	CmsTlsCertDigest    string `yaml:"cms-tls-cert-sha384" mapstructure:"cms-tls-cert-sha384"`
	PollIntervalMinutes int    `yaml:"poll-interval-minutes" mapstructure:"poll-interval-minutes"`

	Log                commConfig.LogConfig     `yaml:"log"`
	IHUB               commConfig.ServiceConfig `yaml:"ihub"`
	AttestationService AttestationConfig        `yaml:"attestation-service" mapstructure:"attestation-service"`
	Endpoint           Endpoint                 `yaml:"end-point" mapstructure:"end-point"`
	TLS                commConfig.TLSCertConfig `yaml:"tls"`
}

type AttestationConfig struct {
	HVSBaseURL  string `yaml:"hvs-base-url" mapstructure:"hvs-base-url"`
	SHVSBaseURL string `yaml:"shvs-base-url" mapstructure:"shvs-base-url"`
}

type Endpoint struct {
	Type     string `yaml:"type" mapstructure:"type"`
	URL      string `yaml:"url" mapstructure:"url"`
	CRDName  string `yaml:"crd-name" mapstructure:"crd-name"`
	Token    string `yaml:"token" mapstructure:"token"`
	CertFile string `yaml:"cert-file" mapstructure:"cert-file"`
}

// this function sets the configure file name and type
func init() {
	viper.SetConfigName(constants.ConfigFile)
	viper.SetConfigType("yaml")
	viper.AddConfigPath(constants.ConfigDir)
}

// SaveConfiguration method used to save the configuration
func (c *Configuration) SaveConfiguration(filename string) error {
	configFile, err := os.OpenFile(filename, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0600)
	if err != nil {
		return errors.Wrap(err, "Failed to create config file")
	}
	defer func() {
		derr := configFile.Close()
		if derr != nil {
			log.WithError(derr).Error("Error closing file")
		}
	}()
	err = yaml.NewEncoder(configFile).Encode(c)
	if err != nil {
		return errors.Wrap(err, "Failed to encode config structure")
	}

	if err := os.Chmod(filename, 0640); err != nil {
		return errors.Wrap(err, "Failed to apply permissions to config file")
	}
	return nil
}

// LoadConfiguration method used to load the configuration
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
