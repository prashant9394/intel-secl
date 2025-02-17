/*
 * Copyright (C) 2020 Intel Corporation
 * SPDX-License-Identifier: BSD-3-Clause
 */
package directory

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"

	"github.com/google/uuid"
	"github.com/intel-secl/intel-secl/v5/pkg/kbs/domain/models"
	commErr "github.com/intel-secl/intel-secl/v5/pkg/lib/common/err"
	"github.com/pkg/errors"
)

type KeyStore struct {
	dir string
}

func NewKeyStore(dir string) *KeyStore {
	return &KeyStore{dir}
}

func (ks *KeyStore) Create(key *models.KeyAttributes) (*models.KeyAttributes, error) {
	defaultLog.Trace("directory/key_store:Create() Entering")
	defer defaultLog.Trace("directory/key_store:Create() Leaving")

	bytes, err := json.Marshal(key)
	if err != nil {
		return nil, errors.Wrap(err, "directory/key_store:Create() Failed to marshal key attributes")
	}

	err = ioutil.WriteFile(filepath.Join(ks.dir, key.ID.String()), bytes, 0600)
	if err != nil {
		return nil, errors.Wrap(err, "directory/key_store:Create() Failed to store key attributes in file")
	}

	return key, nil
}

func (ks *KeyStore) Retrieve(id uuid.UUID) (*models.KeyAttributes, error) {
	defaultLog.Trace("directory/key_store:Retrieve() Entering")
	defer defaultLog.Trace("directory/key_store:Retrieve() Leaving")

	bytes, err := ioutil.ReadFile(filepath.Join(ks.dir, id.String()))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, errors.New(commErr.RecordNotFound)
		} else {
			return nil, errors.Wrapf(err, "directory/key_store:Retrieve() Unable to read key file : %s", id.String())
		}
	}

	var key models.KeyAttributes
	err = json.Unmarshal(bytes, &key)
	if err != nil {
		return nil, errors.Wrap(err, "directory/key_store:Retrieve() Failed to unmarshal key attributes")
	}

	return &key, nil
}

func (ks *KeyStore) Delete(id uuid.UUID) error {
	defaultLog.Trace("directory/key_store:Delete() Entering")
	defer defaultLog.Trace("directory/key_store:Delete() Leaving")

	if err := os.Remove(filepath.Join(ks.dir, id.String())); err != nil {
		if os.IsNotExist(err) {
			return errors.New(commErr.RecordNotFound)
		} else {
			return errors.Wrapf(err, "directory/key_store:Delete() Unable to remove key file : %s", id.String())
		}
	}

	return nil
}

func (ks *KeyStore) Search(criteria *models.KeyFilterCriteria) ([]models.KeyAttributes, error) {
	defaultLog.Trace("directory/key_store:Search() Entering")
	defer defaultLog.Trace("directory/key_store:Search() Leaving")

	var keys = []models.KeyAttributes{}
	keyFiles, err := ioutil.ReadDir(ks.dir)
	if err != nil {
		return nil, errors.Wrapf(err, "directory/key_store:Search() Error in reading the keys directory : %s", ks.dir)
	}

	for _, keyFile := range keyFiles {
		filename, err := uuid.Parse(keyFile.Name())
		if err != nil {
			return nil, errors.Wrapf(err, "directory/key_store:Search() Error in parsing key file name : %s", keyFile.Name())
		}
		key, err := ks.Retrieve(filename)
		if err != nil {
			return nil, errors.Wrapf(err, "directory/key_store:Search() Error in retrieving key from file : %s", keyFile.Name())
		}

		keys = append(keys, *key)
	}

	if len(keys) > 0 {
		keys = filterKeys(keys, criteria)
	}

	return keys, nil
}

// helper function to filter the keys based on given filter criteria.
func filterKeys(keys []models.KeyAttributes, criteria *models.KeyFilterCriteria) []models.KeyAttributes {
	defaultLog.Trace("directory/key_store:filterKeys() Entering")
	defer defaultLog.Trace("directory/key_store:filterKeys() Leaving")

	if criteria == nil || reflect.DeepEqual(*criteria, models.KeyFilterCriteria{}) {
		return keys
	}

	// Algorithm filter
	if criteria.Algorithm != "" {
		var filteredKeys []models.KeyAttributes
		for _, key := range keys {
			if key.Algorithm == criteria.Algorithm {
				filteredKeys = append(filteredKeys, key)
			}
		}
		keys = filteredKeys
	}

	// KeyLength filter
	if criteria.KeyLength != 0 {
		var filteredKeys []models.KeyAttributes
		for _, key := range keys {
			if key.KeyLength == criteria.KeyLength {
				filteredKeys = append(filteredKeys, key)
			}
		}
		keys = filteredKeys
	}

	// CurveType filter
	if criteria.CurveType != "" {
		var filteredKeys []models.KeyAttributes
		for _, key := range keys {
			if key.CurveType == criteria.CurveType {
				filteredKeys = append(filteredKeys, key)
			}
		}
		keys = filteredKeys
	}

	// TransferPolicyId filter
	if criteria.TransferPolicyId != uuid.Nil {
		var filteredKeys []models.KeyAttributes
		for _, key := range keys {
			if key.TransferPolicyId == criteria.TransferPolicyId {
				filteredKeys = append(filteredKeys, key)
			}
		}
		keys = filteredKeys
	}

	return keys
}
