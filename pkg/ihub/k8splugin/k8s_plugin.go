/*
 * Copyright (C) 2022 Intel Corporation
 * SPDX-License-Identifier: BSD-3-Clause
 */
package k8splugin

import (
	"bytes"
	"crypto"
	"crypto/sha512"
	"encoding/json"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/intel-secl/intel-secl/v5/pkg/ihub/util"

	"github.com/intel-secl/intel-secl/v5/pkg/clients/k8s"
	vsPlugin "github.com/intel-secl/intel-secl/v5/pkg/ihub/attestationPlugin"
	"github.com/intel-secl/intel-secl/v5/pkg/ihub/config"
	"github.com/intel-secl/intel-secl/v5/pkg/ihub/constants"
	types "github.com/intel-secl/intel-secl/v5/pkg/ihub/model"
	model "github.com/intel-secl/intel-secl/v5/pkg/model/k8s"

	"io/ioutil"
	"net/http"
	"net/url"

	commonLog "github.com/intel-secl/intel-secl/v5/pkg/lib/common/log"

	"github.com/Waterdrips/jwt-go"
	"github.com/google/uuid"
	"github.com/pkg/errors"

	"encoding/base64"
)

// KubernetesDetails for getting hosts and updating CRD
type KubernetesDetails struct {
	Config             *config.Configuration
	AuthToken          string
	HostDetailsMap     map[string]types.HostDetails
	PrivateKey         crypto.PrivateKey
	PublicKeyBytes     []byte
	K8sClient          *k8s.Client
	TrustedCAsStoreDir string
	SamlCertFilePath   string
}

var (
	log            = commonLog.GetDefaultLogger()
	osRegexEpcSize = regexp.MustCompile(constants.RegexEpcSize)
)

// GetHosts Getting Hosts From Kubernetes
func GetHosts(k8sDetails *KubernetesDetails) error {
	log.Trace("k8splugin/k8s_plugin:GetHosts() Entering")
	defer log.Trace("k8splugin/k8s_plugin:GetHosts() Leaving")
	conf := k8sDetails.Config
	urlPath := conf.Endpoint.URL + constants.KubernetesNodesAPI
	log.Debugf("k8splugin/k8s_plugin:GetHosts() URL to get the Hosts : %s", urlPath)

	parsedUrl, err := url.Parse(urlPath)
	if err != nil {
		return errors.Wrap(err, "k8splugin/k8s_plugin:GetHosts() : Unable to parse the url")
	}

	res, err := k8sDetails.K8sClient.SendRequest(&k8s.RequestParams{
		Method: http.MethodGet,
		URL:    parsedUrl,
		Body:   nil,
	})
	if err != nil {
		return errors.Wrap(err, "k8splugin/k8s_plugin:GetHosts() : Error in getting the Hosts from kubernetes")
	}

	defer func() {
		derr := res.Body.Close()
		if derr != nil {
			log.WithError(derr).Error("Error closing response")
		}
	}()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return errors.Wrap(err, "k8splugin/k8s_plugin:GetHosts() : Error in Reading the Response")
	}

	var hostResponse model.HostResponse
	err = json.Unmarshal(body, &hostResponse)
	if err != nil {
		return errors.Wrap(err, "k8splugin/k8s_plugin:GetHosts() : Error in Unmarshaling the response")
	}

	hostDetailMap := make(map[string]types.HostDetails)

	for _, items := range hostResponse.Items {

		isMaster := false

		for _, taints := range items.Spec.Taints {
			if taints.Key == "node-role.kubernetes.io/master" {
				isMaster = true
				break
			}
		}
		if !isMaster {
			var hostDetails types.HostDetails
			sysID := items.Status.NodeInfo.SystemID
			hostDetails.HostID, _ = uuid.Parse(sysID)

			for _, addr := range items.Status.Addresses {

				if addr.Type == "InternalIP" {
					hostDetails.HostIP = addr.Address
				}

				if addr.Type == "Hostname" {
					hostDetails.HostName = addr.Address
				}
			}

			hostDetailMap[hostDetails.HostIP] = hostDetails
		}

	}
	k8sDetails.HostDetailsMap = hostDetailMap
	return nil
}

// FilterHostReports Get Filtered Host Reports from HVS
func FilterHostReports(k8sDetails *KubernetesDetails, hostDetails *types.HostDetails, trustedCaDir, samlCertPath string) error {

	log.Trace("k8splugin/k8s_plugin:FilterHostReports() Entering")
	defer log.Trace("k8splugin/k8s_plugin:FilterHostReports() Leaving")

	samlReport, err := vsPlugin.GetHostReports(hostDetails.HostID.String(), k8sDetails.Config, trustedCaDir, samlCertPath)
	if err != nil {
		return errors.Wrap(err, "k8splugin/k8s_plugin:FilterHostReports() : Error in getting the host report")
	}

	trustMap := make(map[string]string)
	hardwareFeaturesMap := make(map[string]string)
	assetTagsMap := make(map[string]string)

	for _, as := range samlReport.Attribute {

		if strings.HasPrefix(as.Name, "TAG") {
			assetTagsMap[as.Name] = as.AttributeValue
		}
		if strings.HasPrefix(as.Name, "TRUST") {
			trustMap[as.Name] = as.AttributeValue
		}
		if strings.HasPrefix(as.Name, "FEATURE") {
			hardwareFeaturesMap[as.Name] = as.AttributeValue
		}

	}

	log.Debugf("k8splugin/k8s_plugin:FilterHostReports() Setting Values for Host: %s", hostDetails.HostID.String())

	overAllTrust, _ := strconv.ParseBool(trustMap["TRUST_OVERALL"])
	hostDetails.AssetTags = assetTagsMap
	hostDetails.Trust = trustMap
	hostDetails.HardwareFeatures = hardwareFeaturesMap
	hostDetails.Trusted = overAllTrust
	hostDetails.ValidTo = samlReport.Subject.NotOnOrAfter

	return nil
}

// GetSignedTrustReport Creates a Signed trust-report based on the host details
func GetSignedTrustReport(hostList model.Host, k8sDetails *KubernetesDetails, attestationType string) (string, error) {
	log.Trace("k8splugin/k8s_plugin:GetSignedTrustReport() Entering")
	defer log.Trace("k8splugin/k8s_plugin:GetSignedTrustReport() Leaving")

	hash := sha512.New384()
	_, err := hash.Write(k8sDetails.PublicKeyBytes)
	if err != nil {
		return "", errors.Wrap(err, "k8splugin/k8s_plugin:GetSignedTrustReport() : Error in getting digest of Public key")
	}
	sha384 := base64.StdEncoding.EncodeToString(hash.Sum(nil))

	var token *jwt.Token
	if attestationType == "HVS" {
		token = jwt.NewWithClaims(jwt.SigningMethodRS384, model.HvsHostTrustReport{
			AssetTags:        hostList.AssetTags,
			HardwareFeatures: hostList.HardwareFeatures,
			Trusted:          hostList.Trusted,
			HvsTrustValidTo:  *hostList.HvsTrustValidTo,
		})
	} else if attestationType == "SGX" {
		token = jwt.NewWithClaims(jwt.SigningMethodRS384, model.SgxHostTrustReport{
			SgxSupported:    hostList.SgxSupported,
			SgxEnabled:      hostList.SgxEnabled,
			FlcEnabled:      hostList.FlcEnabled,
			EpcSize:         hostList.EpcSize,
			TcbUpToDate:     hostList.TcbUpToDate,
			SgxTrustValidTo: *hostList.SgxTrustValidTo,
		})
	} else {
		return "", errors.Errorf("k8splugin/k8s_plugin:GetSignedTrustReport() : AttestationType \"%s\" not supported", attestationType)
	}

	token.Header["kid"] = sha384

	// Create the JWT string
	tokenString, err := token.SignedString(k8sDetails.PrivateKey)
	if err != nil {
		return "", errors.Wrap(err, "k8splugin/k8s_plugin:GetSignedTrustReport() : Error in Getting the signed token")
	}

	return tokenString, nil

}

// UpdateCRD Updates the Kubernetes CRD with details from the host report
func UpdateCRD(k8sDetails *KubernetesDetails) error {

	log.Trace("k8splugin/k8s_plugin:UpdateCRD() Entering")
	defer log.Trace("k8splugin/k8s_plugin:UpdateCRD() Leaving")
	k8sConfig := k8sDetails.Config
	crdName := k8sConfig.Endpoint.CRDName
	urlPath := k8sConfig.Endpoint.URL + constants.KubernetesCRDAPI + crdName

	parsedUrl, err := url.Parse(urlPath)
	if err != nil {
		return errors.Wrap(err, "k8splugin/k8s_plugin:UpdateCRD() : Unable to parse the url")
	}
	res, err := k8sDetails.K8sClient.SendRequest(&k8s.RequestParams{
		Method: http.MethodGet,
		URL:    parsedUrl,
		Body:   nil,
	})
	if err != nil {
		return errors.Wrap(err, "k8splugin/k8s_plugin:UpdateCRD() : Error in fetching the kubernetes CRD")
	}
	var crdResponse model.CRD
	if res.StatusCode == http.StatusOK {
		defer func() {
			derr := res.Body.Close()
			if derr != nil {
				log.WithError(derr).Error("Error closing response")
			}
		}()
		body, err := ioutil.ReadAll(res.Body)
		if err != nil {
			return errors.Wrap(err, "k8splugin/k8s_plugin:UpdateCRD() : Error in Reading Response body")
		}

		err = json.Unmarshal(body, &crdResponse)
		if err != nil {
			return errors.Wrap(err, "k8splugin/k8s_plugin:UpdateCRD() : Error in Unmarshalling the CRD Reponse")
		}

		log.Debug("k8splugin/k8s_plugin:UpdateCRD() PUT Call to be made")

		crdResponse.Spec.HostList, err = populateHostDetailsInCRD(k8sDetails)
		if err != nil {
			return errors.Wrap(err, "k8splugin/k8s_plugin:UpdateCRD() : Error populating crd")
		}
		err = PutCRD(k8sDetails, &crdResponse)
		if err != nil {
			return errors.Wrap(err, "k8splugin/k8s_plugin:UpdateCRD() : Error in Updating CRD")
		}
	} else {
		log.Debug("k8splugin/k8s_plugin:UpdateCRD() POST Call to be made")

		crdResponse.APIVersion = constants.KubernetesCRDAPIVersion
		crdResponse.Kind = constants.KubernetesCRDKind
		crdResponse.Metadata.Name = crdName
		crdResponse.Metadata.Namespace = constants.KubernetesMetaDataNameSpace
		crdResponse.Spec.HostList, err = populateHostDetailsInCRD(k8sDetails)
		if err != nil {
			return errors.Wrap(err, "k8splugin/k8s_plugin:UpdateCRD() : Error populating crd")
		}
		log.Debugf("k8splugin/k8s_plugin:UpdateCRD() Printing the spec hostList : %v", crdResponse.Spec.HostList)
		err := PostCRD(k8sDetails, &crdResponse)
		if err != nil {
			return errors.Wrap(err, "k8splugin/k8s_plugin:UpdateCRD() : Error in posting CRD")
		}

	}
	return nil
}

func populateHostDetailsInCRD(k8sDetails *KubernetesDetails) ([]model.Host, error) {
	var hostList []model.Host

	for key := range k8sDetails.HostDetailsMap {

		reportHostDetails := k8sDetails.HostDetailsMap[key]
		var host model.Host
		host.HostName = reportHostDetails.HostName
		t := time.Now().UTC()
		host.Updated = new(time.Time)
		*host.Updated = t
		if reportHostDetails.AgentType != "sgx" {
			host.AssetTags = reportHostDetails.AssetTags
			host.HardwareFeatures = reportHostDetails.HardwareFeatures
			host.Trusted = new(bool)
			*host.Trusted = reportHostDetails.Trusted
			host.HvsTrustValidTo = new(time.Time)
			*host.HvsTrustValidTo = reportHostDetails.ValidTo
			signedtrustReport, err := GetSignedTrustReport(host, k8sDetails, "HVS")
			if err != nil {
				return nil, errors.Wrap(err, "k8splugin/k8s_plugin:populateHostDetailsInCRD() : Error in Getting SignedTrustReport")
			}
			host.HvsSignedTrustReport = signedtrustReport

		}
		if reportHostDetails.AgentType == "sgx" || reportHostDetails.AgentType == "both" {
			host.EpcSize = strings.Replace(reportHostDetails.EpcSize, " ", "", -1)
			host.FlcEnabled = strconv.FormatBool(reportHostDetails.FlcEnabled)
			host.SgxEnabled = strconv.FormatBool(reportHostDetails.SgxEnabled)
			host.SgxSupported = strconv.FormatBool(reportHostDetails.SgxSupported)
			host.TcbUpToDate = reportHostDetails.TcbUpToDate
			host.SgxTrustValidTo = new(time.Time)
			*host.SgxTrustValidTo = reportHostDetails.ValidTo
			signedtrustReport, err := GetSignedTrustReport(host, k8sDetails, "SGX")
			if err != nil {
				return nil, errors.Wrap(err, "k8splugin/k8s_plugin:populateHostDetailsInCRD() : Error in Getting SignedTrustReport")
			}
			host.SgxSignedTrustReport = signedtrustReport
		}

		hostList = append(hostList, host)
	}
	return hostList, nil
}

// PutCRD PUT request call to update existing CRD
func PutCRD(k8sDetails *KubernetesDetails, crd *model.CRD) error {

	log.Trace("k8splugin/k8s_plugin:PutCRD() Entering")
	defer log.Trace("k8splugin/k8s_plugin:PutCRD() Leaving")

	k8sConfig := k8sDetails.Config
	crdName := k8sConfig.Endpoint.CRDName
	urlPath := k8sConfig.Endpoint.URL + constants.KubernetesCRDAPI + crdName

	crdJson, err := json.Marshal(crd)
	if err != nil {
		return errors.Wrap(err, "k8splugin/k8s_plugin:PutCRD() Error in Creating JSON object")
	}

	payload := bytes.NewReader(crdJson)

	parsedUrl, err := url.Parse(urlPath)
	if err != nil {
		return errors.Wrap(err, "k8splugin/k8s_plugin:PutCRD() : Unable to parse the url")
	}

	res, err := k8sDetails.K8sClient.SendRequest(&k8s.RequestParams{
		Method:            http.MethodPut,
		URL:               parsedUrl,
		Body:              payload,
		AdditionalHeaders: map[string]string{"Content-Type": "application/json"},
	})

	if err != nil {
		return errors.Wrap(err, "k8splugin/k8s_plugin:PutCRD() Error in creating CRD")
	}

	defer func() {
		derr := res.Body.Close()
		if derr != nil {
			log.WithError(derr).Error("Error closing response")
		}
	}()

	return nil
}

// PostCRD POST request call to create new CRD
func PostCRD(k8sDetails *KubernetesDetails, crd *model.CRD) error {

	log.Trace("k8splugin/k8s_plugin:PostCRD() Starting")
	defer log.Trace("k8splugin/k8s_plugin:PostCRD() Leaving")
	k8sConfig := k8sDetails.Config
	crdName := k8sConfig.Endpoint.CRDName
	urlPath := k8sConfig.Endpoint.URL + constants.KubernetesCRDAPI + crdName

	crdJSON, err := json.Marshal(crd)
	if err != nil {
		return errors.Wrap(err, "k8splugin/k8s_plugin:PostCRD(): Error in Creating JSON object")
	}
	payload := bytes.NewReader(crdJSON)

	parsedUrl, err := url.Parse(urlPath)
	if err != nil {
		return errors.Wrap(err, "k8splugin/k8s_plugin:PostCRD() : Unable to parse the url")
	}

	_, err = k8sDetails.K8sClient.SendRequest(&k8s.RequestParams{
		Method:            http.MethodPost,
		URL:               parsedUrl,
		Body:              payload,
		AdditionalHeaders: map[string]string{"Content-Type": "application/json"},
	})

	if err != nil {
		return errors.Wrap(err, "k8splugin/k8s_plugin: PostCRD() : Error in creating CRD")
	}

	return nil
}

// SendDataToEndPoint pushes host trust data to Kubernetes
func SendDataToEndPoint(kubernetes KubernetesDetails) error {

	log.Trace("k8splugin/k8s_plugin:SendDataToEndPoint() Entering")
	defer log.Trace("k8splugin/k8s_plugin:SendDataToEndPoint() Leaving")

	var sgxData types.PlatformDataSGX

	log.Debug("k8splugin/k8s_plugin:SendDataToEndPoint() Fetching hosts from Kubernetes")
	err := GetHosts(&kubernetes)
	if err != nil {
		return errors.Wrap(err, "k8splugin/k8s_plugin:SendDataToEndPoint() Error in getting the Hosts from kubernetes")
	}

	log.Infof("k8splugin/k8s_plugin:SendDataToEndPoint() Fetched %d hosts from Kubernetes", len(kubernetes.HostDetailsMap))
	for key := range kubernetes.HostDetailsMap {
		hvsFail := true
		shvsFail := true

		hostDetails := kubernetes.HostDetailsMap[key]

		if kubernetes.Config.AttestationService.HVSBaseURL != "" {
			log.Debugf("k8splugin/k8s_plugin:SendDataToEndPoint() Fetching TrustReport for host %s from HVS", hostDetails.HostID.String())
			err := FilterHostReports(&kubernetes, &hostDetails, kubernetes.TrustedCAsStoreDir, kubernetes.SamlCertFilePath)
			if err != nil {
				log.WithError(err).Warnf("k8splugin/k8s_plugin:SendDataToEndPoint() Could not get TrustReport for host %s from HVS", hostDetails.HostID.String())
			} else {
				hvsFail = false
				// mark Trust Agent as running on this host
				hostDetails.AgentType = "ta"
			}
		}
		if kubernetes.Config.AttestationService.SHVSBaseURL != "" {
			log.Debugf("k8splugin/k8s_plugin:SendDataToEndPoint() Fetching PlatformData for host %s from SHVS", hostDetails.HostName)
			platformData, err := vsPlugin.GetHostPlatformDataSGX(hostDetails.HostName, kubernetes.Config, kubernetes.TrustedCAsStoreDir)
			if err != nil {
				log.WithError(err).Warnf("k8splugin/k8s_plugin:SendDataToEndPoint() Could not get PlatformData for host %s from SHVS", hostDetails.HostName)
			} else {
				shvsFail = false
				// mark SGX agent as running on this host
				hostDetails.AgentType = "sgx"

				err = json.Unmarshal(platformData, &sgxData)
				if err != nil {
					log.WithError(err).Error("k8splugin/k8s_plugin:SendDataToEndPoint() SGX Platform data unmarshal failed")
					continue
				}

				// need to validate contents of EpcSize
				if !osRegexEpcSize.MatchString(sgxData[0].EpcSize) {
					log.WithError(err).Error("k8splugin/k8s_plugin:SendDataToEndPoint() Invalid EPC Size value")
					continue
				}
				hostDetails.EpcSize = sgxData[0].EpcSize
				hostDetails.FlcEnabled = sgxData[0].FlcEnabled
				hostDetails.SgxEnabled = sgxData[0].SgxEnabled
				hostDetails.SgxSupported = sgxData[0].SgxSupported
				hostDetails.TcbUpToDate = strconv.FormatBool(sgxData[0].TcbUpToDate)
				util.EvaluateValidTo(sgxData[0].ValidTo, kubernetes.Config.PollIntervalMinutes)
				hostDetails.ValidTo = sgxData[0].ValidTo

			}
		}
		if !hvsFail && !shvsFail {
			// both SGX agent and Trust agent are running on same host
			hostDetails.AgentType = "both"
		}
		// cannot find this host in HVS or SHVS, remove host from map
		if hvsFail && shvsFail {
			delete(kubernetes.HostDetailsMap, key)
		} else {
			kubernetes.HostDetailsMap[key] = hostDetails
		}
	}

	if len(kubernetes.HostDetailsMap) > 0 {
		log.Debug("Pushing CRDs to Kubernetes")
		err = UpdateCRD(&kubernetes)
		if err != nil {
			return errors.Wrap(err, "k8splugin/k8s_plugin:SendDataToEndPoint() Error in Updating CRDs for Kubernetes")
		}
		log.Infof("k8splugin/k8s_plugin:SendDataToEndPoint() Pushed CRDs to Kubernetes for %d hosts", len(kubernetes.HostDetailsMap))
	}
	return nil
}
