/*
 * Copyright (C) 2020 Intel Corporation
 * SPDX-License-Identifier: BSD-3-Clause
 */
package router

import (
	"github.com/gorilla/mux"
	"github.com/intel-secl/intel-secl/v5/pkg/kbs/constants"
	"github.com/intel-secl/intel-secl/v5/pkg/kbs/controllers"
	"github.com/intel-secl/intel-secl/v5/pkg/kbs/directory"
	"github.com/intel-secl/intel-secl/v5/pkg/lib/common/validation"
	"net/http"
)

//setTpmIdentityCertRoutes registers routes to perform TpmIdentityCertificate CRUD operations
func setTpmIdentityCertRoutes(router *mux.Router) *mux.Router {
	defaultLog.Trace("router/tpm_identity_certificates:setTpmIdentityCertRoutes() Entering")
	defer defaultLog.Trace("router/tpm_identity_certificates:setTpmIdentityCertRoutes() Leaving")

	certStore := directory.NewCertificateStore(constants.TpmIdentityCertsDir)
	tpmIdentityCertController := controllers.NewCertificateController(certStore)
	certIdExpr := "/tpm-identity-certificates/" + validation.IdReg

	router.Handle("/tpm-identity-certificates", ErrorHandler(permissionsHandler(JsonResponseHandler(tpmIdentityCertController.Import),
		[]string{constants.TpmIdentityCertCreate}))).Methods(http.MethodPost)

	router.Handle(certIdExpr, ErrorHandler(permissionsHandler(JsonResponseHandler(tpmIdentityCertController.Retrieve),
		[]string{constants.TpmIdentityCertRetrieve}))).Methods(http.MethodGet)

	router.Handle(certIdExpr, ErrorHandler(permissionsHandler(ResponseHandler(tpmIdentityCertController.Delete),
		[]string{constants.TpmIdentityCertDelete}))).Methods(http.MethodDelete)

	router.Handle("/tpm-identity-certificates", ErrorHandler(permissionsHandler(JsonResponseHandler(tpmIdentityCertController.Search),
		[]string{constants.TpmIdentityCertSearch}))).Methods(http.MethodGet)

	return router
}
