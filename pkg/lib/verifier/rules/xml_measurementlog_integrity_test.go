/*
 * Copyright (C) 2020 Intel Corporation
 * SPDX-License-Identifier: BSD-3-Clause
*/
package rules

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/xml"
	"github.com/google/uuid"
	"github.com/intel-secl/intel-secl/v3/pkg/hvs/constants/verifier-rules-and-faults"
	"github.com/intel-secl/intel-secl/v3/pkg/lib/host-connector/types"
	"github.com/intel-secl/intel-secl/v3/pkg/lib/host-connector/util"
	ta "github.com/intel-secl/intel-secl/v3/pkg/model/ta"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestXmlMeasurementLogIntegrityNoFault(t *testing.T) {

	// create the rule
	var testExpectedMeasurement ta.Measurement
	err := xml.Unmarshal([]byte(testIntegrityMeasurementsXml), &testExpectedMeasurement)
	assert.NoError(t, err)

	rule, err := NewXmlMeasurementLogIntegrity(uuid.MustParse(testExpectedMeasurement.Uuid), testExpectedMeasurement.Label, testExpectedMeasurement.CumulativeHash)
	assert.NoError(t, err)

    // create the manifest that contains the measurement xml and the
    // pcr event log with the correct cumulative measurement
	hostManifest := types.HostManifest {
		MeasurementXmls: []string{testIntegrityMeasurementsXml},
	}

    eventLogEntry := types.EventLogEntry {
		PcrIndex: types.PCR15,
		PcrBank: types.SHA256,
		EventLogs: []types.EventLog {
			{
				DigestType: util.EVENT_LOG_DIGEST_SHA256,
                Value: getSha256String(testExpectedMeasurement.CumulativeHash),
                Label: testExpectedMeasurement.Label + "-" + testExpectedMeasurement.Uuid,
            },
        },
    }

    hostManifest.PcrManifest.PcrEventLogMap.Sha256EventLogs = append(hostManifest.PcrManifest.PcrEventLogMap.Sha256EventLogs, eventLogEntry)

	// apply the manifest to the rule and expect no faults/trusted
	result, err := rule.Apply(&hostManifest)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Trusted)
	assert.Equal(t, len(result.Faults), 0)
}

func TestXmlMeasurementLogIntegrityXmlEventLogMissingFault(t *testing.T) {

    // create the rule
	var testExpectedMeasurement ta.Measurement
	err := xml.Unmarshal([]byte(testIntegrityMeasurementsXml), &testExpectedMeasurement)
	assert.NoError(t, err)

	rule, err := NewXmlMeasurementLogIntegrity(uuid.MustParse(testExpectedMeasurement.Uuid), testExpectedMeasurement.Label, testExpectedMeasurement.CumulativeHash)
	assert.NoError(t, err)

	// provide an empty manifest without xml and expect FaultXmlMeasurementLogMissing/untrusted
    result, err := rule.Apply(&types.HostManifest{})
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 1, len(result.Faults))
	assert.Equal(t, constants.FaultXmlMeasurementLogMissing, result.Faults[0].Name)
	t.Logf("Fault description: %s", result.Faults[0].Description)
}

func TestXmlMeasurementLogIntegrityXmlMeasurementLogInvalidFault(t *testing.T) {

    // create the rule
    var testExpectedMeasurement ta.Measurement
    err := xml.Unmarshal([]byte(testIntegrityMeasurementsXml), &testExpectedMeasurement)
    assert.NoError(t, err)

    rule, err := NewXmlMeasurementLogIntegrity(uuid.MustParse(testExpectedMeasurement.Uuid), testExpectedMeasurement.Label, testExpectedMeasurement.CumulativeHash)
	assert.NoError(t, err)

    // provide the rule a manifest with invalid xml and expect FaultXmlMeasurementLogInvalid
    hostManifest := types.HostManifest {
		MeasurementXmls: []string{"invalid xml"},
	}

    result, err := rule.Apply(&hostManifest)
    assert.NoError(t, err)
    assert.NotNil(t, result)
    assert.Equal(t, 1, len(result.Faults))
    assert.Equal(t, constants.FaultXmlMeasurementLogInvalid, result.Faults[0].Name)
    t.Logf("Fault description: %s", result.Faults[0].Description)
}

func TestXmlMeasurementLogIntegrityXmlMissingFromBadLabel(t *testing.T) {

    // create the rule
    var testExpectedMeasurement ta.Measurement
    err := xml.Unmarshal([]byte(testIntegrityMeasurementsXml), &testExpectedMeasurement)
    assert.NoError(t, err)

    rule, err := NewXmlMeasurementLogIntegrity(uuid.MustParse(testExpectedMeasurement.Uuid), testExpectedMeasurement.Label, testExpectedMeasurement.CumulativeHash)
	assert.NoError(t, err)

    // create a manifest with a different "label" than the flavor and exepct FaultXmlMeasurementLogMissing
    var invalidMeasurements ta.Measurement
    err = xml.Unmarshal([]byte(testIntegrityMeasurementsXml), &invalidMeasurements)
    assert.NoError(t, err)

    invalidMeasurements.Label = "invalidlabel"
	invalidMeasurementsXml, err := xml.Marshal(invalidMeasurements)
    assert.NoError(t, err)
    
	hostManifest := types.HostManifest {
		MeasurementXmls: []string{string(invalidMeasurementsXml)},
	}

    result, err := rule.Apply(&hostManifest)
    assert.NoError(t, err)
    assert.NotNil(t, result)
    assert.Equal(t, 1, len(result.Faults))
    assert.Equal(t, constants.FaultXmlMeasurementLogMissing, result.Faults[0].Name)
    t.Logf("Fault description: %s", result.Faults[0].Description)
}

func TestXmlMeasurementLogIntegrityValueMismatchFromInvalidActualHash(t *testing.T) {

    // create the rule
    var testExpectedMeasurement ta.Measurement
    err := xml.Unmarshal([]byte(testIntegrityMeasurementsXml), &testExpectedMeasurement)
    assert.NoError(t, err)

    rule, err := NewXmlMeasurementLogIntegrity(uuid.MustParse(testExpectedMeasurement.Uuid), testExpectedMeasurement.Label, testExpectedMeasurement.CumulativeHash)
	assert.NoError(t, err)

    // change the manifest's CumulativeHash to not match the flavor and expect
    // FaultXmlMeasurementValueMismatch
    var invalidMeasurements ta.Measurement
    err = xml.Unmarshal([]byte(testIntegrityMeasurementsXml), &invalidMeasurements)
    assert.NoError(t, err)

    invalidMeasurements.CumulativeHash = "00000000000000000000"
	invalidMeasurementsXml, err := xml.Marshal(invalidMeasurements)
    assert.NoError(t, err)
    
	hostManifest := types.HostManifest {
		MeasurementXmls: []string{string(invalidMeasurementsXml)},
	}

    result, err := rule.Apply(&hostManifest)
    assert.NoError(t, err)
    assert.NotNil(t, result)
    assert.Equal(t, 1, len(result.Faults))
    assert.Equal(t, constants.FaultXmlMeasurementValueMismatch, result.Faults[0].Name)
    t.Logf("Fault description: %s", result.Faults[0].Description)
}

func TestXmlMeasurementLogIntegrityValueMismatchFromInvalidReplay(t *testing.T) {

    // create the rule
    var testExpectedMeasurement ta.Measurement
    err := xml.Unmarshal([]byte(testIntegrityMeasurementsXml), &testExpectedMeasurement)
    assert.NoError(t, err)

    rule, err := NewXmlMeasurementLogIntegrity(uuid.MustParse(testExpectedMeasurement.Uuid), testExpectedMeasurement.Label, testExpectedMeasurement.CumulativeHash)
	assert.NoError(t, err)

    // change one of the event measurements so that the calculated hash is different
    // than the expected hash (expect FaultXmlMeasurementValueMismatch)
    var invalidMeasurements ta.Measurement
    err = xml.Unmarshal([]byte(testIntegrityMeasurementsXml), &invalidMeasurements)
    assert.NoError(t, err)

    invalidMeasurements.File[0].Value = "00000000000000000000"
	invalidMeasurementsXml, err := xml.Marshal(invalidMeasurements)
    assert.NoError(t, err)
    
	hostManifest := types.HostManifest {
		MeasurementXmls: []string{string(invalidMeasurementsXml)},
	}

    result, err := rule.Apply(&hostManifest)
    assert.NoError(t, err)
    assert.NotNil(t, result)
    assert.Equal(t, 1, len(result.Faults))
    assert.Equal(t, constants.FaultXmlMeasurementValueMismatch, result.Faults[0].Name)
    t.Logf("Fault description: %s", result.Faults[0].Description)
}

//TODO: Fix this along with the Rule fix
/*
func TestXmlMeasurementLogIntegrityValueMismatchFromInvalidPcrEventLog(t *testing.T) {

    // create the rule
    var testExpectedMeasurement ta.Measurement
    err := xml.Unmarshal([]byte(testIntegrityMeasurementsXml), &testExpectedMeasurement)
    assert.NoError(t, err)

    rule, err := NewXmlMeasurementLogIntegrity(uuid.MustParse(testExpectedMeasurement.Uuid), testExpectedMeasurement.Label, testExpectedMeasurement.CumulativeHash)
	assert.NoError(t, err)

    // create a manifest with valid event log xml but a pcr event log with
    // an invalid mesurement (expect FaultXmlMeasurementValueMismatch)
	hostManifest := types.HostManifest {
		MeasurementXmls: []string{testIntegrityMeasurementsXml},
	}

    eventLogEntry := types.EventLogEntry {
		PcrIndex: types.PCR15,
		PcrBank: types.SHA256,
		EventLogs: []types.EventLog {
			{
				DigestType: util.EVENT_LOG_DIGEST_SHA256,
                Value: "0000000000000000000000000000000000", // ==> NOT RIGHT
                Label: testExpectedMeasurement.Label + "-" + testExpectedMeasurement.Uuid,
            },
        },
    }

    hostManifest.PcrManifest.PcrEventLogMap.Sha256EventLogs = append(hostManifest.PcrManifest.PcrEventLogMap.Sha256EventLogs, eventLogEntry)

    result, err := rule.Apply(&hostManifest)
    assert.NoError(t, err)
    assert.NotNil(t, result)
    assert.Equal(t, 1, len(result.Faults))
    assert.Equal(t, constants.FaultXmlMeasurementValueMismatch, result.Faults[0].Name)
    t.Logf("Fault description: %s", result.Faults[0].Description)
}
*/

func TestXmlMeasurementLogIntegrityValueMismatchFromMissingPcrEventLog(t *testing.T) {

    // create the rule
    var testExpectedMeasurement ta.Measurement
    err := xml.Unmarshal([]byte(testIntegrityMeasurementsXml), &testExpectedMeasurement)
    assert.NoError(t, err)

    rule, err := NewXmlMeasurementLogIntegrity(uuid.MustParse(testExpectedMeasurement.Uuid), testExpectedMeasurement.Label, testExpectedMeasurement.CumulativeHash)
	assert.NoError(t, err)

    // create a manifest with valid event log xml but without pcr event log
    // and expect FaultPcrValueMissing
	hostManifest := types.HostManifest {
		MeasurementXmls: []string{testIntegrityMeasurementsXml},
	}

    result, err := rule.Apply(&hostManifest)
    assert.NoError(t, err)
    assert.NotNil(t, result)
    assert.Equal(t, 1, len(result.Faults))
    assert.Equal(t, constants.FaultPcrEventLogMissing, result.Faults[0].Name)
    t.Logf("Fault description: %s", result.Faults[0].Description)
}

func TestXmlMeasurementLogIntegrityValueMismatchFromMissingPcrEventLabel(t *testing.T) {

    // create the rule
    var testExpectedMeasurement ta.Measurement
    err := xml.Unmarshal([]byte(testIntegrityMeasurementsXml), &testExpectedMeasurement)
    assert.NoError(t, err)

    rule, err := NewXmlMeasurementLogIntegrity(uuid.MustParse(testExpectedMeasurement.Uuid), testExpectedMeasurement.Label, testExpectedMeasurement.CumulativeHash)
	assert.NoError(t, err)

    // create a manifest with a pcr event log that does not contain a
    // matching 'label'  (expect FaultXmlMeasurementValueMismatch)
	hostManifest := types.HostManifest {
		MeasurementXmls: []string{testIntegrityMeasurementsXml},
	}

    eventLogEntry := types.EventLogEntry {
		PcrIndex: types.PCR15,
		PcrBank: types.SHA256,
		EventLogs: []types.EventLog {
			{
				DigestType: util.EVENT_LOG_DIGEST_SHA256,
                Value: getSha256String(testExpectedMeasurement.CumulativeHash),
                Label: "invalid labor", // ==> won't match the flavor
            },
        },
    }

    hostManifest.PcrManifest.PcrEventLogMap.Sha256EventLogs = append(hostManifest.PcrManifest.PcrEventLogMap.Sha256EventLogs, eventLogEntry)

    result, err := rule.Apply(&hostManifest)
    assert.NoError(t, err)
    assert.NotNil(t, result)
    assert.Equal(t, 1, len(result.Faults))
    assert.Equal(t, constants.FaultXmlMeasurementValueMismatch, result.Faults[0].Name)
    t.Logf("Fault description: %s", result.Faults[0].Description)
}

func getSha256String(existingHash string) string {

    existingBytes, _ := hex.DecodeString(existingHash)

    h := sha256.New()
    h.Write(existingBytes)
    newBytes := h.Sum(nil)

    return hex.EncodeToString(newBytes)
}
