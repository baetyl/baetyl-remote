module github.com/baetyl/baetyl-remote

replace (
	github.com/docker/docker => github.com/docker/engine v0.0.0-20191007211215-3e077fc8667a
	github.com/opencontainers/runc => github.com/opencontainers/runc v1.0.1-0.20190307181833-2b18fe1d885e
)

go 1.13

require (
	github.com/256dpi/gomqtt v0.12.3
	github.com/aws/aws-sdk-go v1.25.36
	github.com/baetyl/baetyl v0.0.0-20191118112140-b09fbcad9795
	github.com/baidubce/bce-sdk-go v0.9.5
	github.com/docker/distribution v2.7.1+incompatible
	github.com/docker/go-units v0.4.0
	github.com/golang/mock v1.3.1 // indirect
	github.com/panjf2000/ants v1.3.0
	github.com/stretchr/objx v0.2.0 // indirect
	github.com/stretchr/testify v1.4.0
	github.com/vektra/mockery v0.0.0-20181123154057-e78b021dcbb5 // indirect
	golang.org/x/net v0.0.0-20191119073136-fc4aabc6c914 // indirect
	golang.org/x/tools v0.0.0-20191121040551-947d4aa89328 // indirect
	gopkg.in/yaml.v2 v2.2.7
)
