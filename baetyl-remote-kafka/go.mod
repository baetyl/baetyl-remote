module github.com/baetyl/baetyl-remote/baetyl-remote-kafka

go 1.13

replace (
	github.com/docker/docker => github.com/docker/engine v0.0.0-20191007211215-3e077fc8667a
	github.com/opencontainers/runc => github.com/opencontainers/runc v1.0.1-0.20190307181833-2b18fe1d885e
)

require (
	github.com/256dpi/gomqtt v0.12.2
	github.com/baetyl/baetyl v0.0.0-20191029040747-94cee1e78d62
	github.com/segmentio/kafka-go v0.3.4
	golang.org/x/tools v0.0.0-20190524140312-2c0ae7006135
)
