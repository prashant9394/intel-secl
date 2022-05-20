/*
 * Copyright (C) 2022 Intel Corporation
 * SPDX-License-Identifier: BSD-3-Clause
 */
package attestationPlugin

import (
	"io/ioutil"
	"testing"

	"github.com/google/uuid"

	"github.com/intel-secl/intel-secl/v5/pkg/ihub/config"
	testutility "github.com/intel-secl/intel-secl/v5/pkg/ihub/test"
	commConfig "github.com/intel-secl/intel-secl/v5/pkg/lib/common/config"
)

func TestGetHostReportsTee(t *testing.T) {
	server := testutility.MockServer(t)
	defer server.Close()

	output, err := ioutil.ReadFile("../../ihub/test/resources/tee_platform_data.json")
	if err != nil {
		t.Log("attestationPlugin/tee_plugin_test:TestGetHostReportsTee(): Unable to read file", err)
	}

	sgxHostHardwareID := uuid.MustParse("00b61da0-5ada-e811-906e-00163566263e")
	type args struct {
		hostHardwareId uuid.UUID
		config         *config.Configuration
	}
	tests := []struct {
		name    string
		args    args
		want    []byte
		wantErr bool
	}{
		{
			name: "Valid Test: get-tee-host-platform-data",
			args: args{
				hostHardwareId: sgxHostHardwareID,
				config: &config.Configuration{
					AASBaseUrl: server.URL + "/aas/v1",
					IHUB: commConfig.ServiceConfig{
						Username: "test@user",
						Password: "test@password",
					},
					AttestationService: config.AttestationConfig{
						FDSBaseURL: server.URL + "/fds/v1/",
					},
				},
			},
			want:    output,
			wantErr: false,
		},
		{
			name: "Invalid Test1: get-tee-host-platform-data",
			args: args{
				hostHardwareId: sgxHostHardwareID,
				config:         &config.Configuration{},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetHostPlatformData(tt.args.hostHardwareId, tt.args.config, sampleRootCertDirPath)
			if (err != nil) != tt.wantErr {
				t.Errorf("TestGetHostReportsTee() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if len(got) != 1 && !tt.wantErr {
				t.Errorf("TestGetHostReportsTee(): Could not retrieve host platform data")
			}
		})
	}
}

func Test_initializeFDSClient(t *testing.T) {
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
			name: "Valid Test: initialize-fds-client",
			args: args{
				certDirectory: "",
				con: &config.Configuration{
					AASBaseUrl: server.URL + "/aas/v1",
					IHUB: commConfig.ServiceConfig{
						Username: "test@user",
						Password: "test@password",
					},
					AttestationService: config.AttestationConfig{
						FDSBaseURL: server.URL + "/fds/v1",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "Invalid Test2: initialize-fds-client",
			args: args{
				certDirectory: "",
				con: &config.Configuration{
					AASBaseUrl: server.URL + "/aas/v1",
					IHUB: commConfig.ServiceConfig{
						Username: "test@user",
						Password: "test@password",
					},
					AttestationService: config.AttestationConfig{
						FDSBaseURL: server.URL + "/fds\v1",
					},
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := initializeFDSClient(tt.args.con, tt.args.certDirectory)
			if (err != nil) != tt.wantErr {
				t.Errorf("attestationPlugin/sgx_plugin_test:initializeFDSClient() Error in initializing client :error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
