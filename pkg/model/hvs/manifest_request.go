/*
 *  Copyright (C) 2020 Intel Corporation
 *  SPDX-License-Identifier: BSD-3-Clause
 */

package hvs

import (
	"encoding/xml"
	"github.com/google/uuid"
	model "github.com/intel-secl/intel-secl/v5/pkg/model/ta"
)

type ManifestRequest struct {
	XMLName xml.Name `xml:"ManifestRequest"`
	// swagger:strfmt uuid
	HostId           uuid.UUID      `xml:"hostId,omitempty"`
	ConnectionString string         `xml:"connectionString"`
	FlavorGroupNames []string       `xml:"flavorgroupNames,omitempty"`
	Manifest         model.Manifest `xml:"Manifest"`
}
