/*
 * Copyright (C) 2021 Intel Corporation
 * SPDX-License-Identifier: BSD-3-Clause
 */

package tasks

import (
	"fmt"
	"github.com/intel-secl/intel-secl/v5/pkg/kbs/constants"
	"io"
	"strconv"
	"strings"

	"github.com/intel-secl/intel-secl/v5/pkg/kbs/config"
	commConfig "github.com/intel-secl/intel-secl/v5/pkg/lib/common/config"
	"github.com/intel-secl/intel-secl/v5/pkg/lib/common/setup"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

type UpdateServiceConfig struct {
	ServerConfig  commConfig.ServerConfig
	DefaultPort   int
	AASBaseUrl    string
	APSBaseUrl    string
	CustomToken   string
	AppConfig     **config.Configuration
	ConsoleWriter io.Writer
}

const envHelpPrompt = "Following environment variables are required for update-service-config setup:"

var allowedKeyManagers = map[string]bool{"kmip": true}

var envHelp = map[string]string{
	"AAS_BASE_URL":               "AAS Base URL",
	"LOG_LEVEL":                  "Log level",
	"LOG_MAX_LENGTH":             "Max length of log statement",
	"LOG_ENABLE_STDOUT":          "Enable console log",
	"KMIP_SERVER_IP":             "IP of KMIP server",
	"KMIP_SERVER_PORT":           "PORT of KMIP server",
	"KMIP_HOSTNAME":              "HOSTNAME of KMIP server",
	"KMIP_USERNAME":              "USERNAME of KMIP server",
	"KMIP_PASSWORD":              "PASSWORD of KMIP server",
	"KMIP_CLIENT_CERT_PATH":      "KMIP Client certificate path",
	"KMIP_CLIENT_KEY_PATH":       "KMIP Client key path",
	"KMIP_ROOT_CERT_PATH":        "KMIP Root Certificate path",
	"SERVER_PORT":                "The Port on which Server Listens to",
	"SERVER_READ_TIMEOUT":        "Request Read Timeout Duration in Seconds",
	"SERVER_READ_HEADER_TIMEOUT": "Request Read Header Timeout Duration in Seconds",
	"SERVER_WRITE_TIMEOUT":       "Request Write Timeout Duration in Seconds",
	"SERVER_IDLE_TIMEOUT":        "Request Idle Timeout in Seconds",
	"SERVER_MAX_HEADER_BYTES":    "Max Length Of Request Header in Bytes ",
}

func (uc UpdateServiceConfig) Run() error {
	log.Trace("tasks/update_config:Run() Entering")
	defer log.Trace("tasks/update_config:Run() Leaving")

	if uc.AASBaseUrl == "" {
		return errors.New("KBS configuration not provided: AAS_BASE_URL is not set")
	}

	if uc.ServerConfig.Port < 1024 ||
		uc.ServerConfig.Port > 65535 {
		uc.ServerConfig.Port = uc.DefaultPort
	}

	(*uc.AppConfig).Server = uc.ServerConfig
	(*uc.AppConfig).AASBaseUrl = uc.AASBaseUrl

	if !strings.HasSuffix(uc.APSBaseUrl, "/") {
		uc.APSBaseUrl = uc.APSBaseUrl + "/"
	}
	(*uc.AppConfig).APSBaseUrl = uc.APSBaseUrl
	(*uc.AppConfig).CustomToken = uc.CustomToken

	(*uc.AppConfig).Log = commConfig.LogConfig{
		MaxLength:    viper.GetInt(commConfig.LogMaxLength),
		EnableStdout: viper.GetBool(commConfig.LogEnableStdout),
		Level:        viper.GetString(commConfig.LogLevel),
	}
	(*uc.AppConfig).EndpointURL = viper.GetString(config.EndpointUrl)
	(*uc.AppConfig).Kmip = config.KmipConfig{
		Version:                   viper.GetString(config.KmipVersion),
		ServerIP:                  viper.GetString(config.KmipServerIP),
		ServerPort:                viper.GetString(config.KmipServerPort),
		Hostname:                  viper.GetString(config.KmipHostname),
		Username:                  viper.GetString(config.KmipUsername),
		Password:                  viper.GetString(config.KmipPassword),
		ClientKeyFilePath:         viper.GetString(config.KmipClientKeyPath),
		ClientCertificateFilePath: viper.GetString(config.KmipClientCertPath),
		RootCertificateFilePath:   viper.GetString(config.KmipRootCertPath),
	}
	(*uc.AppConfig).KeyManager = viper.GetString(config.KeyManager)
	return nil
}

func (uc UpdateServiceConfig) Validate() error {
	if uc.AASBaseUrl == "" {
		return errors.New("KBS configuration not provided: AAS_BASE_URL is not set")
	}
	if (*uc.AppConfig).Server.Port < 1024 ||
		(*uc.AppConfig).Server.Port > 65535 {
		return errors.New("Configured port is not valid")
	}
	if (*uc.AppConfig).Log.MaxLength < constants.MinLogLengthLimit || (*uc.AppConfig).Log.MaxLength > constants.MaxLogLengthLimit {
		return errors.New("tasks/update_service_config:Validate() Configured Log Length not valid. Please specify value within " + strconv.Itoa(constants.MinLogLengthLimit) + " and " + strconv.Itoa(constants.MaxLogLengthLimit))
	}
	if _, validInput := allowedKeyManagers[strings.ToLower((*uc.AppConfig).KeyManager)]; !validInput {
		return errors.New("Invalid value provided for KEY_MANAGER. Value should be kmip")
	}
	return nil
}
func (uc UpdateServiceConfig) PrintHelp(w io.Writer) {
	setup.PrintEnvHelp(w, envHelpPrompt, "", envHelp)
	fmt.Fprintln(w, "")
}

func (uc UpdateServiceConfig) SetName(n, e string) {
}
