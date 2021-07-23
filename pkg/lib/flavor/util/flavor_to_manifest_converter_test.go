/*
 * Copyright (C) 2020 Intel Corporation
 * SPDX-License-Identifier: BSD-3-Clause
 */
package util

import (
	"encoding/json"
	"github.com/intel-secl/intel-secl/v5/pkg/model/hvs"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"os"
	"testing"
)

const (
	PFFlavorJson string = "../test/resources/PfFlavor.json"
)

var (
	fmc FlavorToManifestConverter
)

func TestFlavorToManifestConverter_GetManifestXML(t *testing.T) {
	var myflavor hvs.Flavor

	// load flavor
	flavorFile, _ := os.Open(PFFlavorJson)
	flavorBytes, _ := ioutil.ReadAll(flavorFile)
	_ = json.Unmarshal(flavorBytes, &myflavor)

	// convert to manifest
	mmanifest := fmc.GetManifestFromFlavor(myflavor)
	assert.NotNil(t, mmanifest)
	t.Log(mmanifest)

}
