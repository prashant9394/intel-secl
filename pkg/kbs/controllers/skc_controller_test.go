/*
 * Copyright (C) 2020 Intel Corporation
 * SPDX-License-Identifier: BSD-3-Clause
 */

package controllers_test

import (
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/gorilla/mux"
	"github.com/intel-secl/intel-secl/v4/pkg/kbs/config"
	"github.com/intel-secl/intel-secl/v4/pkg/kbs/controllers"
	"github.com/intel-secl/intel-secl/v4/pkg/kbs/domain/mocks"
	"github.com/intel-secl/intel-secl/v4/pkg/kbs/keymanager"
	"github.com/intel-secl/intel-secl/v4/pkg/kbs/kmipclient"
	kbsRoutes "github.com/intel-secl/intel-secl/v4/pkg/kbs/router"
	consts "github.com/intel-secl/intel-secl/v4/pkg/lib/common/constants"
	"github.com/intel-secl/intel-secl/v4/pkg/lib/common/crypt"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
)

var _ = Describe("SKCKeyTransferController", func() {
	var router *mux.Router
	var server *ghttp.Server
	var w *httptest.ResponseRecorder
	var keyStore *mocks.MockKeyStore
	var policyStore *mocks.MockKeyTransferPolicyStore
	var remoteManager *keymanager.RemoteManager
	var skcController *controllers.SKCController
	var kbsConfig *config.Configuration

	certPem, _ := ioutil.ReadFile(skcClientCertPath)
	cert, _ := crypt.GetCertFromPem(certPem)
	cs := tls.ConnectionState{PeerCertificates: []*x509.Certificate{cert}}

	mockClient := kmipclient.NewMockKmipClient()
	keyManager := keymanager.NewKmipManager(mockClient)

	BeforeEach(func() {
		router = mux.NewRouter()
		server = ghttp.NewServer()
		keyStore = mocks.NewFakeKeyStore()
		policyStore = mocks.NewFakeKeyTransferPolicyStore()
		kbsConfig = &config.Configuration{
			AASApiUrl: "http://" + server.Addr() + "/aas/",
			KBS: config.KBSConfig{
				UserName: KBSServiceUserName,
				Password: KBSServicePassword,
			},
			Skc: config.SKCConfig{
				StmLabel: "SGX",
				SQVSUrl:  "http://" + server.Addr() + "/svs/v1",
			},
		}

		remoteManager = keymanager.NewRemoteManager(keyStore, keyManager, endpointUrl)
		skcController = controllers.NewSKCController(remoteManager, policyStore, kbsConfig, trustedCaCertsDir)
		setupServer(server)
	})

	AfterEach(func() {
		server.Close()
	})

	// Specs for HTTP GET to "/dhsm2-transfer"
	Describe("Transfers an existing Key", func() {
		Context("Provide a valid Transfer request", func() {
			BeforeEach(func() {
				sessionController := controllers.NewSessionController(kbsConfig, trustedCaCertsDir)
				router.Handle("/session", kbsRoutes.ErrorHandler(kbsRoutes.JsonResponseHandler(sessionController.Create))).Methods("POST")
				sessionJson := `{
							"challenge_type": "SGX",
							"challenge": "MTRjZmNlZDEtMDNlZS00YTY4LThiNTAtNmQ0NTY0MjNiMDc4",
							"quote": "AQAAAAAAAACLEwAAAQAAAAEAAAADAAAAAAEAAOwGAAAtLS0tLUJFR0lOIENFUlRJRklDQVRFLS0tLS0KTUlJRTlEQ0NCSnFnQXdJQkFnSVVMMnV3N0VOc3FSTnFGL1dsRFFGcVc4WVFUN1F3Q2dZSUtvWkl6ajBFQXdJd2NERWlNQ0FHQTFVRQpBd3daU1c1MFpXd2dVMGRZSUZCRFN5QlFiR0YwWm05eWJTQkRRVEVhTUJnR0ExVUVDZ3dSU1c1MFpXd2dRMjl5Y0c5eVlYUnBiMjR4CkZEQVNCZ05WQkFjTUMxTmhiblJoSUVOc1lYSmhNUXN3Q1FZRFZRUUlEQUpEUVRFTE1Ba0dBMVVFQmhNQ1ZWTXdIaGNOTWpBeE1URXoKTURFME1UQXlXaGNOTWpjeE1URXpNREUwTVRBeVdqQndNU0l3SUFZRFZRUUREQmxKYm5SbGJDQlRSMWdnVUVOTElFTmxjblJwWm1sagpZWFJsTVJvd0dBWURWUVFLREJGSmJuUmxiQ0JEYjNKd2IzSmhkR2x2YmpFVU1CSUdBMVVFQnd3TFUyRnVkR0VnUTJ4aGNtRXhDekFKCkJnTlZCQWdNQWtOQk1Rc3dDUVlEVlFRR0V3SlZVekJaTUJNR0J5cUdTTTQ5QWdFR0NDcUdTTTQ5QXdFSEEwSUFCQkpZMytWbVBCNG0KN0lRWEhiTTA3Wlp4WXBvbTJUTnNWSnpZd2UrVytzWWtXV1FncFlrQ0hQd2RNa2ZBU05JM2pld01pOWM1SDFlODV2RCtMSEYwalAragpnZ01RTUlJREREQWZCZ05WSFNNRUdEQVdnQlJaSTlPblNxaGpWQzQ1Y0szZ0R3Y3JWeVFxdHpCdkJnTlZIUjhFYURCbU1HU2dZcUJnCmhsNW9kSFJ3Y3pvdkwzTmllQzVoY0drdWRISjFjM1JsWkhObGNuWnBZMlZ6TG1sdWRHVnNMbU52YlM5elozZ3ZZMlZ5ZEdsbWFXTmgKZEdsdmJpOTJNeTl3WTJ0amNtdy9ZMkU5Y0d4aGRHWnZjbTBtWlc1amIyUnBibWM5WkdWeU1CMEdBMVVkRGdRV0JCUkxTMWE5QWhCMwptYjhDUjJZVkpwa3hPZUxxVWpBT0JnTlZIUThCQWY4RUJBTUNCc0F3REFZRFZSMFRBUUgvQkFJd0FEQ0NBamtHQ1NxR1NJYjRUUUVOCkFRU0NBaW93Z2dJbU1CNEdDaXFHU0liNFRRRU5BUUVFRVBnYXlXbVNZMFF6V1lVZG14Vnc0Und3Z2dGakJnb3Foa2lHK0UwQkRRRUMKTUlJQlV6QVFCZ3NxaGtpRytFMEJEUUVDQVFJQkFqQVFCZ3NxaGtpRytFMEJEUUVDQWdJQkFqQVFCZ3NxaGtpRytFMEJEUUVDQXdJQgpBREFRQmdzcWhraUcrRTBCRFFFQ0JBSUJBREFRQmdzcWhraUcrRTBCRFFFQ0JRSUJBREFRQmdzcWhraUcrRTBCRFFFQ0JnSUJBREFRCkJnc3Foa2lHK0UwQkRRRUNCd0lCQURBUUJnc3Foa2lHK0UwQkRRRUNDQUlCQURBUUJnc3Foa2lHK0UwQkRRRUNDUUlCQURBUUJnc3EKaGtpRytFMEJEUUVDQ2dJQkFEQVFCZ3NxaGtpRytFMEJEUUVDQ3dJQkFEQVFCZ3NxaGtpRytFMEJEUUVDREFJQkFEQVFCZ3NxaGtpRworRTBCRFFFQ0RRSUJBREFRQmdzcWhraUcrRTBCRFFFQ0RnSUJBREFRQmdzcWhraUcrRTBCRFFFQ0R3SUJBREFRQmdzcWhraUcrRTBCCkRRRUNFQUlCQURBUUJnc3Foa2lHK0UwQkRRRUNFUUlCQ2pBZkJnc3Foa2lHK0UwQkRRRUNFZ1FRQWdJQUFBQUFBQUFBQUFBQUFBQUEKQURBUUJnb3Foa2lHK0UwQkRRRURCQUlBQURBVUJnb3Foa2lHK0UwQkRRRUVCQVlRWUdvQUFBQXdEd1lLS29aSWh2aE5BUTBCQlFvQgpBVEFlQmdvcWhraUcrRTBCRFFFR0JCQTc5SWlaYWlhbXJSSW5zdWEwcFRqak1FUUdDaXFHU0liNFRRRU5BUWN3TmpBUUJnc3Foa2lHCitFMEJEUUVIQVFFQi96QVFCZ3NxaGtpRytFMEJEUUVIQWdFQkFEQVFCZ3NxaGtpRytFMEJEUUVIQXdFQi96QUtCZ2dxaGtqT1BRUUQKQWdOSUFEQkZBaUVBdXNreWlrcFMvT1RHWG5tckJDY25QUXlnWElocWVjbDY4NExWaWJQNEpvZ0NJQ0xhKzZ6Uzg2c1paVFZwTGxtWApIS1ZoTkxTRExLVUwvb3pOa1N3eVZta2YKLS0tLS1FTkQgQ0VSVElGSUNBVEUtLS0tLQEAAc7FpcesytSmdaNdgy+Gkj+5B/Q0V2vfWXb6viRziyZNRo9KgEK9tRwi50f7uOTv5i0xkOV2QoH+iIRtTtKRlFU2Rdh8JJn+B3nvg3vxNdYcCLkT7w8oY0olLq7GfwRtsaP5z9855Q2ucYiXNBCH5YfyCjX8xCttY1jt8r8GI5kS4rzGnz78upVoW1doMet2ONLxbsFUB6Zn3VgGrQ5xxgiAAuK9wtvQTfVLCgjioYsp/NEDeSeyojkxKw6/r0+Phlec2BoTnSqYTnfyYL/G1qjSV6ifXsgrchL+oFnkLTeJ3lC8B+QVKrqVboZ+EIPULIbvjmUSe0674fayRnUXYR8DAAIAAAAAAAUACgCTmnIz95xMqZQKDbOVfwYHKkxL2d+pXquq/oS/FUYgvwAAAAACAgACAgAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAHAAAAAAAAAOcAAAAAAAAAffC36BW9S0r0EjkDjQSnQNrM8L60EqIFbI2QC0W2If0AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAM0XHFaUHGzklpC0VfaR2cigTC5D4KTTD3UvpShcfuV/AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAJglV6uZR4R63bkOU579zXVg5QpysXJqSzgu8MbRaBuvAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAADUEAAAJlWoc9gx/u9YZnn06FBk5Y9C12bJPrSgZ1D4GUahEAUS2dql45hOxhpRFI3JRY+G6zm7R1qQkICoGNZaDlDOZ/KBkjJ+GTyEo45QiUUHLRHV8hk9ByGrLhedkfgFDDeWk0Jd1IQDRVY/PQmApEhIlAuwW4A4pAhuZasAxBVC1/QCAgACAgAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAVAAAAAAAAAOcAAAAAAAAAYNha8ovo0cQKCNmLAJ1fiswThKOFz0YIAOR4eR0al5wAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAIxPV3XXllA+lhN/d8aKgpoAVqyN7XAUCwgbCUSQxXv/AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAQAFAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAFjXV3GsuftfSiE0iCo+i7/kYJc7ozHiv6mzAuciE3RAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAADiC3/hSeunDDyw1IOxJy8WD7n0OamNDOvV0XXeKwrPYKFAP4lYzpp4kSUdJiG0/9CG2GYULCd6lTyyG0U+LQrgIAAAAQIDBAUGBwgJCgsMDQ4PEBESExQVFhcYGRobHB0eHwUAbA4AAC0tLS0tQkVHSU4gQ0VSVElGSUNBVEUtLS0tLQpNSUlFOURDQ0JKcWdBd0lCQWdJVUwydXc3RU5zcVJOcUYvV2xEUUZxVzhZUVQ3UXdDZ1lJS29aSXpqMEVBd0l3Y0RFaU1DQUdBMVVFCkF3d1pTVzUwWld3Z1UwZFlJRkJEU3lCUWJHRjBabTl5YlNCRFFURWFNQmdHQTFVRUNnd1JTVzUwWld3Z1EyOXljRzl5WVhScGIyNHgKRkRBU0JnTlZCQWNNQzFOaGJuUmhJRU5zWVhKaE1Rc3dDUVlEVlFRSURBSkRRVEVMTUFrR0ExVUVCaE1DVlZNd0hoY05NakF4TVRFegpNREUwTVRBeVdoY05NamN4TVRFek1ERTBNVEF5V2pCd01TSXdJQVlEVlFRRERCbEpiblJsYkNCVFIxZ2dVRU5MSUVObGNuUnBabWxqCllYUmxNUm93R0FZRFZRUUtEQkZKYm5SbGJDQkRiM0p3YjNKaGRHbHZiakVVTUJJR0ExVUVCd3dMVTJGdWRHRWdRMnhoY21FeEN6QUoKQmdOVkJBZ01Ba05CTVFzd0NRWURWUVFHRXdKVlV6QlpNQk1HQnlxR1NNNDlBZ0VHQ0NxR1NNNDlBd0VIQTBJQUJCSlkzK1ZtUEI0bQo3SVFYSGJNMDdaWnhZcG9tMlROc1ZKell3ZStXK3NZa1dXUWdwWWtDSFB3ZE1rZkFTTkkzamV3TWk5YzVIMWU4NXZEK0xIRjBqUCtqCmdnTVFNSUlERERBZkJnTlZIU01FR0RBV2dCUlpJOU9uU3FoalZDNDVjSzNnRHdjclZ5UXF0ekJ2QmdOVkhSOEVhREJtTUdTZ1lxQmcKaGw1b2RIUndjem92TDNOaWVDNWhjR2t1ZEhKMWMzUmxaSE5sY25acFkyVnpMbWx1ZEdWc0xtTnZiUzl6WjNndlkyVnlkR2xtYVdOaApkR2x2Ymk5Mk15OXdZMnRqY213L1kyRTljR3hoZEdadmNtMG1aVzVqYjJScGJtYzlaR1Z5TUIwR0ExVWREZ1FXQkJSTFMxYTlBaEIzCm1iOENSMllWSnBreE9lTHFVakFPQmdOVkhROEJBZjhFQkFNQ0JzQXdEQVlEVlIwVEFRSC9CQUl3QURDQ0Fqa0dDU3FHU0liNFRRRU4KQVFTQ0Fpb3dnZ0ltTUI0R0NpcUdTSWI0VFFFTkFRRUVFUGdheVdtU1kwUXpXWVVkbXhWdzRSd3dnZ0ZqQmdvcWhraUcrRTBCRFFFQwpNSUlCVXpBUUJnc3Foa2lHK0UwQkRRRUNBUUlCQWpBUUJnc3Foa2lHK0UwQkRRRUNBZ0lCQWpBUUJnc3Foa2lHK0UwQkRRRUNBd0lCCkFEQVFCZ3NxaGtpRytFMEJEUUVDQkFJQkFEQVFCZ3NxaGtpRytFMEJEUUVDQlFJQkFEQVFCZ3NxaGtpRytFMEJEUUVDQmdJQkFEQVEKQmdzcWhraUcrRTBCRFFFQ0J3SUJBREFRQmdzcWhraUcrRTBCRFFFQ0NBSUJBREFRQmdzcWhraUcrRTBCRFFFQ0NRSUJBREFRQmdzcQpoa2lHK0UwQkRRRUNDZ0lCQURBUUJnc3Foa2lHK0UwQkRRRUNDd0lCQURBUUJnc3Foa2lHK0UwQkRRRUNEQUlCQURBUUJnc3Foa2lHCitFMEJEUUVDRFFJQkFEQVFCZ3NxaGtpRytFMEJEUUVDRGdJQkFEQVFCZ3NxaGtpRytFMEJEUUVDRHdJQkFEQVFCZ3NxaGtpRytFMEIKRFFFQ0VBSUJBREFRQmdzcWhraUcrRTBCRFFFQ0VRSUJDakFmQmdzcWhraUcrRTBCRFFFQ0VnUVFBZ0lBQUFBQUFBQUFBQUFBQUFBQQpBREFRQmdvcWhraUcrRTBCRFFFREJBSUFBREFVQmdvcWhraUcrRTBCRFFFRUJBWVFZR29BQUFBd0R3WUtLb1pJaHZoTkFRMEJCUW9CCkFUQWVCZ29xaGtpRytFMEJEUUVHQkJBNzlJaVphaWFtclJJbnN1YTBwVGpqTUVRR0NpcUdTSWI0VFFFTkFRY3dOakFRQmdzcWhraUcKK0UwQkRRRUhBUUVCL3pBUUJnc3Foa2lHK0UwQkRRRUhBZ0VCQURBUUJnc3Foa2lHK0UwQkRRRUhBd0VCL3pBS0JnZ3Foa2pPUFFRRApBZ05JQURCRkFpRUF1c2t5aWtwUy9PVEdYbm1yQkNjblBReWdYSWhxZWNsNjg0TFZpYlA0Sm9nQ0lDTGErNnpTODZzWlpUVnBMbG1YCkhLVmhOTFNETEtVTC9vek5rU3d5Vm1rZgotLS0tLUVORCBDRVJUSUZJQ0FURS0tLS0tLS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUNtakNDQWtDZ0F3SUJBZ0lVV1NQVHAwcW9ZMVF1T1hDdDRBOEhLMWNrS3Jjd0NnWUlLb1pJemowRUF3SXcKYURFYU1CZ0dBMVVFQXd3UlNXNTBaV3dnVTBkWUlGSnZiM1FnUTBFeEdqQVlCZ05WQkFvTUVVbHVkR1ZzSUVOdgpjbkJ2Y21GMGFXOXVNUlF3RWdZRFZRUUhEQXRUWVc1MFlTQkRiR0Z5WVRFTE1Ba0dBMVVFQ0F3Q1EwRXhDekFKCkJnTlZCQVlUQWxWVE1CNFhEVEU1TVRBek1URXlNek0wTjFvWERUTTBNVEF6TVRFeU16TTBOMW93Y0RFaU1DQUcKQTFVRUF3d1pTVzUwWld3Z1UwZFlJRkJEU3lCUWJHRjBabTl5YlNCRFFURWFNQmdHQTFVRUNnd1JTVzUwWld3ZwpRMjl5Y0c5eVlYUnBiMjR4RkRBU0JnTlZCQWNNQzFOaGJuUmhJRU5zWVhKaE1Rc3dDUVlEVlFRSURBSkRRVEVMCk1Ba0dBMVVFQmhNQ1ZWTXdXVEFUQmdjcWhrak9QUUlCQmdncWhrak9QUU1CQndOQ0FBUXdwK0xjK1RVQnRnMUgKK1U4SklzTXNiakhqQ2tUdFhiOGpQTTZyMmRodTl6SWJsaERaN0lOZnF0M0l4OFhjRktEOGswTkVYcmtaNjZxSgpYYTFLekxJS280Ry9NSUc4TUI4R0ExVWRJd1FZTUJhQUZPbm9SRkpUTmx4TEdKb1IvRU1ZTEtYY0lJQklNRllHCkExVWRId1JQTUUwd1M2QkpvRWVHUldoMGRIQnpPaTh2YzJKNExXTmxjblJwWm1sallYUmxjeTUwY25WemRHVmsKYzJWeWRtbGpaWE11YVc1MFpXd3VZMjl0TDBsdWRHVnNVMGRZVW05dmRFTkJMbVJsY2pBZEJnTlZIUTRFRmdRVQpXU1BUcDBxb1kxUXVPWEN0NEE4SEsxY2tLcmN3RGdZRFZSMFBBUUgvQkFRREFnRUdNQklHQTFVZEV3RUIvd1FJCk1BWUJBZjhDQVFBd0NnWUlLb1pJemowRUF3SURTQUF3UlFJaEFKMXErRlR6K2dVdVZmQlF1Q2dKc0ZyTDJUVFMKZTFhQlo1M081MlRqRmllNkFpQXJpUGFSYWhVWDlPYTlrR0xsQWNoV1hLVDZqNFJXU1I1MEJxaHJOM1VUNEE9PQotLS0tLUVORCBDRVJUSUZJQ0FURS0tLS0tCi0tLS0tQkVHSU4gQ0VSVElGSUNBVEUtLS0tLQpNSUlDbERDQ0FqbWdBd0lCQWdJVkFPbm9SRkpUTmx4TEdKb1IvRU1ZTEtYY0lJQklNQW9HQ0NxR1NNNDlCQU1DCk1HZ3hHakFZQmdOVkJBTU1FVWx1ZEdWc0lGTkhXQ0JTYjI5MElFTkJNUm93R0FZRFZRUUtEQkZKYm5SbGJDQkQKYjNKd2IzSmhkR2x2YmpFVU1CSUdBMVVFQnd3TFUyRnVkR0VnUTJ4aGNtRXhDekFKQmdOVkJBZ01Ba05CTVFzdwpDUVlEVlFRR0V3SlZVekFlRncweE9URXdNekV3T1RRNU1qRmFGdzAwT1RFeU16RXlNelU1TlRsYU1HZ3hHakFZCkJnTlZCQU1NRVVsdWRHVnNJRk5IV0NCU2IyOTBJRU5CTVJvd0dBWURWUVFLREJGSmJuUmxiQ0JEYjNKd2IzSmgKZEdsdmJqRVVNQklHQTFVRUJ3d0xVMkZ1ZEdFZ1EyeGhjbUV4Q3pBSkJnTlZCQWdNQWtOQk1Rc3dDUVlEVlFRRwpFd0pWVXpCWk1CTUdCeXFHU000OUFnRUdDQ3FHU000OUF3RUhBMElBQkUvNkQvMVdITnJXd1BtTk1JeUJLTVc1Cko2SnpNc2pvNnhQMnZrSzFjZFpHYjFQR1JQL0MvOEVDZ2lEa21rbG16d0x6TGkrMDAwbTdMTHJ0S0pBM29DMmoKZ2I4d2did3dId1lEVlIwakJCZ3dGb0FVNmVoRVVsTTJYRXNZbWhIOFF4Z3NwZHdnZ0Vnd1ZnWURWUjBmQkU4dwpUVEJMb0VtZ1I0WkZhSFIwY0hNNkx5OXpZbmd0WTJWeWRHbG1hV05oZEdWekxuUnlkWE4wWldSelpYSjJhV05sCmN5NXBiblJsYkM1amIyMHZTVzUwWld4VFIxaFNiMjkwUTBFdVpHVnlNQjBHQTFVZERnUVdCQlRwNkVSU1V6WmMKU3hpYUVmeERHQ3lsM0NDQVNEQU9CZ05WSFE4QkFmOEVCQU1DQVFZd0VnWURWUjBUQVFIL0JBZ3dCZ0VCL3dJQgpBVEFLQmdncWhrak9QUVFEQWdOSkFEQkdBaUVBenc5emRVaVVIUE1VZDBDNG14NDFqbEZaa3JNM3k1ZjFsZ25WCk83RmJqT29DSVFDb0d0VW1UNGNYdDdWK3lTSGJKOEhvYjlBYW5wdlhOSDFFUisvZ1pGK29wUT09Ci0tLS0tRU5EIENFUlRJRklDQVRFLS0tLS0K",
							"userData": "ATE0Y2ZjZWQxMDNlZTRhNjg4YjUwNmQ0NTY0MjNiMDc4"
						}`
				req, err := http.NewRequest(
					"POST",
					"/session",
					strings.NewReader(sessionJson),
				)
				req.Header.Set("Accept", consts.HTTPMediaTypeJson)
				req.Header.Set("Content-Type", consts.HTTPMediaTypeJson)
				Expect(err).NotTo(HaveOccurred())
				w = httptest.NewRecorder()
				router.ServeHTTP(w, req)
				Expect(w.Code).To(Equal(http.StatusBadRequest))
			})
			It("Should fail to transfer an existing Key", func() {
				router.Handle("/keys/{id}/dhsm2-transfer", kbsRoutes.ErrorHandler(kbsRoutes.JsonResponseHandler(skcController.TransferApplicationKey))).Methods("GET")
				req, err := http.NewRequest("GET", "/keys/ee37c360-7eae-4250-a677-6ee12adce8e2/dhsm2-transfer", nil)
				req.Header.Set("Accept", consts.HTTPMediaTypeJson)
				req.Header.Set("Accept-Challenge", "SGX")
				req.Header.Set("Session-Id", "SGX:14cfced1-03ee-4a68-8b50-6d456423b078")
				req.TLS = &cs
				Expect(err).NotTo(HaveOccurred())
				w = httptest.NewRecorder()
				router.ServeHTTP(w, req)
				Expect(w.Code).To(Equal(http.StatusInternalServerError))
			})
		})
		Context("Provide a Transfer request without Accept-Challenge Header", func() {
			It("Should fail to transfer an existing Key", func() {
				router.Handle("/keys/{id}/dhsm2-transfer", kbsRoutes.ErrorHandler(kbsRoutes.JsonResponseHandler(skcController.TransferApplicationKey))).Methods("GET")
				req, err := http.NewRequest("GET", "/keys/ee37c360-7eae-4250-a677-6ee12adce8e2/dhsm2-transfer", nil)
				req.Header.Set("Accept", consts.HTTPMediaTypeJson)
				req.Header.Set("Session-Id", "SGX:14cfced1-03ee-4a68-8b50-6d456423b078")
				req.TLS = &cs
				Expect(err).NotTo(HaveOccurred())
				w = httptest.NewRecorder()
				router.ServeHTTP(w, req)
				Expect(w.Code).To(Equal(http.StatusBadRequest))
			})
		})
		Context("Provide a Transfer request without Session-Id Header", func() {
			It("Should fail to transfer an existing Key", func() {
				router.Handle("/keys/{id}/dhsm2-transfer", kbsRoutes.ErrorHandler(kbsRoutes.JsonResponseHandler(skcController.TransferApplicationKey))).Methods("GET")
				req, err := http.NewRequest("GET", "/keys/ee37c360-7eae-4250-a677-6ee12adce8e2/dhsm2-transfer", nil)
				req.Header.Set("Accept", consts.HTTPMediaTypeJson)
				req.Header.Set("Accept-Challenge", "SGX")
				req.TLS = &cs
				Expect(err).NotTo(HaveOccurred())
				w = httptest.NewRecorder()
				router.ServeHTTP(w, req)
				Expect(w.Code).To(Equal(http.StatusUnauthorized))
			})
		})
	})
})
