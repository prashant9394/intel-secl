# Go Workload Service

## Software requirements

- git
- makeself
- `go` version 1.18.8

### Install `go` version 1.18.8

The `Workload Service` requires Go version 1.18.8 that has support for `go modules`. The build was validated with the latest version go1.18.8 of `go`. It is recommended that you use go1.18.8 version of `go`. More recent versions may introduce compatibility issues. You can use the following to install `go`.

```shell
wget https://dl.google.com/go/go1.18.8.linux-amd64.tar.gz
tar -xzf go1.18.8.linux-amd64.tar.gz
sudo mv go /usr/local
export GOROOT=/usr/local/go
export PATH=$GOPATH/bin:$GOROOT/bin:$PATH
```

### Build

```console
> make all
```

Installer Bin will be available in out/wls-*.bin Exportable docker image will be available in out/ as well

### Deploy

```console
> ./wls-*.bin
```

OR

```console
> docker-compose -f dist/docker/docker-compose.yml up
```

### Deployment Config

The table below provides some details on the deployment configuration required in the /root/wls.env. A sample is also provided in the dist/linux path.

Variable               | Data Type      | Required?                   | Default Value                          | Description                                                                      | Example
---------------------- | -------------- | --------------------------- | -------------------------------------- | -------------------------------------------------------------------------------- | -------------------------------------------------------
WLS_LOGLEVEL           | String         | No                          | INFO                                   | Logging level of the Workload Service                                            | Info/Error/Debug
WLS_ENABLE_CONSOLE_LOG | Boolean        | No                          | false                                  | If set to true, logs will printed on console                                     | true
LOG_ENTRY_MAXLENGTH    | Integer        | No                          | 300                                    | Maximum length of a log entry for Workload Service                               | 500
WLS_PORT               | Integer        | No                          | 5000                                   | Listener Port of the Workload Service                                            | 5000
HVS_URL                | URL            | Yes                         | -                                      | Host Verification Service Endpoint                                               | <https://hvs.example.com:8443:/mtwilson/v2/>
CMS_BASE_URL           | URL            | Yes                         | -                                      | Cert Management Service Endpoint                                                 | <https://certservice.example.com:8445:/cms/v1/>
CMS_TLS_CERT_SHA384    | String         | Yes                         | -                                      | Sha384 Hash value of the CMS TLS Certificate - required to validate CMS TLS cert |
AAS_API_URL            | URL            | Yes                         | -                                      | AAS Endpoint                                                                     | <https://authservice.example.com:8444/aas>
BEARER_TOKEN           | JWT Token      | Yes                         | -                                      | JWT token from AAS containing roles required by WLS for setup tasks              |
WLS_SERVICE_USERNAME   | String         | Yes                         | -                                      | Username in AAS which has the relevant roles assigned for WLS                    | admin@wls
WLS_SERVICE_PASSWORD   | String         | Yes                         | -                                      | Password for AAS user account assigned to WLS                                    | wlsAdminPassword
WLS_TLS_CERT_CN        | String         | No                          | WLS TLS Certificate                    | Common Name in WLS TLS Certificate                                               | Acme Inc Enterprise Workload Service Instance
WLS_NOSETUP            | boolean        | No                          | WLS No Setup Flag                      | If set to "true" the setup tasks are skipped, else the setup tasks are skipped   | true/false
SAN_LIST               | CSV of strings | No                          | 127.0.0.1,localhost                    | List of FQDNs to be added on Cert Request to CMS                                 | wls.example.com,workloadserivce.example.com
CERT_PATH              | String         | No                          | /etc/wls/tls-cert.pem     | Filesystem path where the CA certificates will be downloaded from CMS            |
KEY_PATH               | String         | no                          | /etc/wls/tls.key          | Filesystem path where the SAML verification key from HVS will be stored          |

## Manage service

- Start service

  - wls start

- Stop service

  - wls stop

- Status of service

  - wls status
