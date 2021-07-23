/*
 * Copyright (C) 2020 Intel Corporation
 * SPDX-License-Identifier: BSD-3-Clause
 */
package host_connector

import (
	"crypto/tls"
	"crypto/x509"
	"github.com/intel-secl/intel-secl/v5/pkg/hvs/constants"
	cos "github.com/intel-secl/intel-secl/v5/pkg/lib/common/os"
	"net/url"
	"strings"

	client "github.com/intel-secl/intel-secl/v5/pkg/clients/ta"
	"github.com/intel-secl/intel-secl/v5/pkg/lib/host-connector/types"
	"github.com/pkg/errors"
)

type IntelConnectorFactory struct {
	natsServers []string
}

func (icf *IntelConnectorFactory) GetHostConnector(vendorConnector types.VendorConnector, aasApiUrl string,
	trustedCaCerts []x509.Certificate) (HostConnector, error) {

	var taClient client.TAClient

	log.Trace("intel_host_connector_factory:GetHostConnector() Entering")
	defer log.Trace("intel_host_connector_factory:GetHostConnector() Leaving")
	baseURL := vendorConnector.Url
	if !strings.Contains(baseURL, "/v2") {
		baseURL = baseURL + "/v2"
	}

	taApiURL, err := url.Parse(baseURL)
	if err != nil {
		return nil, errors.New("intel_host_connector_factory:GetHostConnector() error retrieving TA API URL")
	}

	if taApiURL.Scheme == "nats" {
		rootCAs, _ := x509.SystemCertPool()
		if rootCAs == nil {
			rootCAs = x509.NewCertPool()
		}

		certs, err := cos.GetDirFileContents(constants.TrustedCaCertsDir, "*.pem")
		if err != nil {
			log.Errorf("client/nats_client:newNatsConnection() Failed to append %q to RootCAs: %v", "/etc/hvs/certs/trustedca/nats-ca.pem", err)
		}

		for _, rootCACert := range certs {
			if ok := rootCAs.AppendCertsFromPEM(rootCACert); !ok {
				log.Info("client/nats_client:newNatsConnection() No certs appended, using system certs only")
			}
		}

		tlsConfig := &tls.Config{
			InsecureSkipVerify: false,
			RootCAs:            rootCAs,
		}
		// in the form: intel:nats://<nats-host-id> (where nats-host-id could be 'foo' or 'host1.intel.com')
		taClient, err = client.NewNatsTAClient(icf.natsServers, taApiURL.Host, tlsConfig, constants.NatsCredentials)
		if err != nil {
			return nil, errors.Wrap(err, "intel_host_connector_factory:GetHostConnector() Could not create nats Trust Agent client")
		}

	} else {

		taClient, err = client.NewTAClient(aasApiUrl,
			taApiURL,
			vendorConnector.Configuration.Username,
			vendorConnector.Configuration.Password,
			trustedCaCerts)

		if err != nil {
			return nil, errors.Wrap(err, "intel_host_connector_factory:GetHostConnector() Could not create Trust Agent client")
		}
	}

	log.Debug("intel_host_connector_factory:GetHostConnector() TA client created")
	return &IntelConnector{taClient}, nil
}
