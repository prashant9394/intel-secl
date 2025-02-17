# Integration Hub

`Integration Hub Service` is a web service that helps in sending updated trust informations to the orchestrator endpoints. Integration Hub fetches attestation details from HVS and updates it to the endpoint orchestrators like Openstack/Kubernetes.
## Key features
- Retrieves attestation details at configured interval from the Host Verification service.
- Pushes attestation details to configured orchestrators e.g OpenStack/Kubernetes


## Build Integration Hub
- Git clone the Mono-Repo which includes Integration Hub
- Run below scripts to build the Integration hub

```shell
git clone https://github.com/intel-secl/intel-secl.git
cd intel-isecl
make ihub-installer
```

### Deploy
```console
> ./ihub-*.bin
```

### Manage service
* Start service
    * ihub start
* Stop service
    * ihub stop
* Status of service
    * ihub status

### Direct dependencies

| Name        | Repo URL                            | Minimum Version Required                          |
| ----------- | ------------------------------------| :------------------------------------------------ |
| jwt-go      | github.com/Waterdrips/jwt-go        | v3.2.1-0.20200915121943-f6506928b72e+incompatible |
| uuid        | github.com/google/uuid              | v1.2.0                                            |
| mux         | github.com/gorilla/mux              | v1.7.4                                            |
| logrus      | github.com/sirupsen/logrus          | v1.7.0                                            |
| goxmldsig   | github.com/russellhaering/goxmldsig | v0.0.0-20180430223755-7acd5e4a6ef7                | 
| errors      | github.com/pkg/errors               | v0.9.1                                            |
| testify     | github.com/stretchr/testify         | v1.6.1	                                        |
| yaml.v3     | gopkg.in/yaml.v3                    | v3.0.1.0                                          |


*Note: All dependencies are listed in go.mod*

# Links
 - Use [Automated Build Steps](https://01.org/intel-secl/documentation/build-installation-scripts) to build all repositories in one go, this will also provide provision to install prerequisites and would handle order and version of dependent repositories.

***Note:** Automated script would install a specific version of the build tools, which might be different than the one you are currently using*
 - [Product Documentation](https://01.org/intel-secl/documentation/intel%C2%AE-secl-dc-product-guide)

