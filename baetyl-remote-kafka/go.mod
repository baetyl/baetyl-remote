module github.com/baetyl/baetyl-remote/baetyl-remote-kafka

go 1.13

replace (
	github.com/docker/docker => github.com/docker/engine v0.0.0-20191007211215-3e077fc8667a
	github.com/opencontainers/runc => github.com/opencontainers/runc v1.0.1-0.20190307181833-2b18fe1d885e
)

require (
	github.com/256dpi/gomqtt v0.12.2
	github.com/Microsoft/go-winio v0.4.15 // indirect
	github.com/StackExchange/wmi v0.0.0-20190523213315-cbe66965904d // indirect
	github.com/baetyl/baetyl v0.0.0-20191029040747-94cee1e78d62
	github.com/frankban/quicktest v1.11.2 // indirect
	github.com/go-ole/go-ole v1.2.4 // indirect
	github.com/segmentio/kafka-go v0.3.4
	github.com/shirou/w32 v0.0.0-20160930032740-bb4de0191aa4 // indirect
	gotest.tools v2.2.0+incompatible // indirect
)
