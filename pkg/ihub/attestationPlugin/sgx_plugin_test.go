/*
 * Copyright (C) 2022 Intel Corporation
 * SPDX-License-Identifier: BSD-3-Clause
*/
package attestationPlugin

import (
	"io/ioutil"
	"reflect"
	"testing"

	"github.com/intel-secl/intel-secl/v5/pkg/clients/sgxhvsclient"

	"github.com/intel-secl/intel-secl/v5/pkg/ihub/config"
	testutility "github.com/intel-secl/intel-secl/v5/pkg/ihub/test"
	commConfig "github.com/intel-secl/intel-secl/v5/pkg/lib/common/config"
)

func TestGetHostReportsSGX(t *testing.T) {
	server := testutility.MockServer(t)
	defer server.Close()

	output, err := ioutil.ReadFile("../../ihub/test/resources/sgx_platform_data.json")
	if err != nil {
		t.Log("attestationPlugin/sgx_plugin_test:TestGetHostReportsSGX(): Unable to read file", err)
	}

	sgxHostName := "localhost"
	type args struct {
		hostIP string
		config *config.Configuration
	}
	tests := []struct {
		name    string
		args    args
		want    []byte
		wantErr bool
	}{
		{
			name: "Valid Test: get-sgx-host-platform-data",
			args: args{
				hostIP: sgxHostName,
				config: &config.Configuration{
					AASBaseUrl: server.URL + "/aas/v1",
					IHUB: commConfig.ServiceConfig{
						Username: "admin@hub",
						Password: "@#!$%",
					},
					AttestationService: config.AttestationConfig{
						SHVSBaseURL: server.URL + "/sgx-hvs/v2/",
					},
				},
			},
			want:    output,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetHostPlatformDataSGX(tt.args.hostIP, tt.args.config, sampleRootCertDirPath)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetHostReportsSGX() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetHostReportsSGX() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_initializeSKCClient(t *testing.T) {
	server := testutility.MockServer(t)
	defer server.Close()

	type args struct {
		con           *config.Configuration
		certDirectory string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{

		{
			name: "Valid Test: initialize-skc-client",
			args: args{
				certDirectory: "",
				con: &config.Configuration{
					AASBaseUrl: server.URL + "/aas/v1",
					IHUB: commConfig.ServiceConfig{
						Username: "admin@hub",
						Password: "@#$%@",
					},
					AttestationService: config.AttestationConfig{
						SHVSBaseURL: server.URL + "/sgx-hvs/v2",
					},
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		SGXClient = &sgxhvsclient.Client{}
		t.Run(tt.name, func(t *testing.T) {
			_, err := initializeSKCClient(tt.args.con, tt.args.certDirectory)
			if (err != nil) != tt.wantErr {
				t.Errorf("attestationPlugin/sgx_plugin_test:initializeSKCClient() Error in initializing client :error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
