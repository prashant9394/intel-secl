/*
 *  Copyright (C) 2020 Intel Corporation
 *  SPDX-License-Identifier: BSD-3-Clause
 */

package router

import (
	"github.com/gorilla/mux"
	"github.com/intel-secl/intel-secl/v5/pkg/authservice/controllers"
	"github.com/intel-secl/intel-secl/v5/pkg/authservice/domain"
	"net/http"
)

func SetRolesRoutes(r *mux.Router, db domain.AASDatabase) *mux.Router {
	defaultLog.Trace("router/roles:SetRolesRoutes() Entering")
	defer defaultLog.Trace("router/roles:SetRolesRoutes() Leaving")

	controller := controllers.RolesController{Database: db}

	r.Handle("/roles", ErrorHandler(ResponseHandler(controller.CreateRole, "application/json"))).Methods(http.MethodPost)
	r.Handle("/roles", ErrorHandler(ResponseHandler(controller.QueryRoles, "application/json"))).Methods(http.MethodGet)
	r.Handle("/roles/{id}", ErrorHandler(ResponseHandler(controller.DeleteRole, ""))).Methods(http.MethodDelete)
	r.Handle("/roles/{id}", ErrorHandler(ResponseHandler(controller.GetRole, "application/json"))).Methods(http.MethodGet)
	r.Handle("/roles/{id}", ErrorHandler(ResponseHandler(controller.UpdateRole, ""))).Methods("PATCH")
	return r
}
