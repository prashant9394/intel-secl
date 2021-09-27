/*
 * Copyright (C) 2020 Intel Corporation
 * SPDX-License-Identifier: BSD-3-Clause
 */
package utils

import (
	"github.com/intel-secl/intel-secl/v5/pkg/lib/common/log"
	"github.com/intel-secl/intel-secl/v5/pkg/model/hvs"
)

var defaultLog = log.GetDefaultLogger()

func CreateFlavorGroupByName(flavorgroupName string) hvs.FlavorGroup {
	defaultLog.Trace("utils/flavor_group:CreateFlavorGroupByName() Entering")
	defer defaultLog.Trace("utils/flavor_group:CreateFlavorGroupByName() Leaving")

	var flavorgroup hvs.FlavorGroup
	flavorgroup.Name = flavorgroupName
	flavorgroup.MatchPolicies = GetAutomaticFlavorMatchPolicy()
	return flavorgroup
}

func GetAutomaticFlavorMatchPolicy() []hvs.FlavorMatchPolicy {
	defaultLog.Trace("utils/flavor_group:GetAutomaticFlavorMatchPolicy() Entering")
	defer defaultLog.Trace("utils/flavor_group:GetAutomaticFlavorMatchPolicy() Leaving")

	var policies []hvs.FlavorMatchPolicy
	policies = append(policies, hvs.NewFlavorMatchPolicy(hvs.FlavorPartPlatform, hvs.NewMatchPolicy(hvs.MatchTypeAnyOf, hvs.FlavorRequired)))
	policies = append(policies, hvs.NewFlavorMatchPolicy(hvs.FlavorPartOs, hvs.NewMatchPolicy(hvs.MatchTypeAnyOf, hvs.FlavorRequired)))
	policies = append(policies, hvs.NewFlavorMatchPolicy(hvs.FlavorPartSoftware, hvs.NewMatchPolicy(hvs.MatchTypeAllOf, hvs.FlavorRequiredIfDefined)))
	policies = append(policies, hvs.NewFlavorMatchPolicy(hvs.FlavorPartAssetTag, hvs.NewMatchPolicy(hvs.MatchTypeLatest, hvs.FlavorRequiredIfDefined)))
	policies = append(policies, hvs.NewFlavorMatchPolicy(hvs.FlavorPartHostUnique, hvs.NewMatchPolicy(hvs.MatchTypeLatest, hvs.FlavorRequiredIfDefined)))

	return policies
}
