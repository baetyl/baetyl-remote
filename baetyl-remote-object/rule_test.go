package main

import (
	"io/ioutil"
	"os"
	"path"
	"testing"
	"time"

	"github.com/baetyl/baetyl-go/v2/context"
	"github.com/stretchr/testify/assert"
)

const (
	caPem = `
-----BEGIN CERTIFICATE-----
MIICjTCCAjKgAwIBAgIIFiYYXpptZ7AwCgYIKoZIzj0EAwIwgawxCzAJBgNVBAYT
AkNOMRAwDgYDVQQIEwdCZWlqaW5nMRkwFwYDVQQHExBIYWlkaWFuIERpc3RyaWN0
MRUwEwYDVQQJEwxCYWlkdSBDYW1wdXMxDzANBgNVBBETBjEwMDA5MzEeMBwGA1UE
ChMVTGludXggRm91bmRhdGlvbiBFZGdlMQ8wDQYDVQQLEwZCQUVUWUwxFzAVBgNV
BAMTDmRlZmF1bHQuMDcyOTAxMCAXDTIwMDcyOTAyMzE1MloYDzIwNzAwNzE3MDIz
MTUyWjCBrDELMAkGA1UEBhMCQ04xEDAOBgNVBAgTB0JlaWppbmcxGTAXBgNVBAcT
EEhhaWRpYW4gRGlzdHJpY3QxFTATBgNVBAkTDEJhaWR1IENhbXB1czEPMA0GA1UE
ERMGMTAwMDkzMR4wHAYDVQQKExVMaW51eCBGb3VuZGF0aW9uIEVkZ2UxDzANBgNV
BAsTBkJBRVRZTDEXMBUGA1UEAxMOZGVmYXVsdC4wNzI5MDEwWTATBgcqhkjOPQIB
BggqhkjOPQMBBwNCAASIpuCgm+W8OIb6njb4K8XCBnuGCNNkGwmFX1S45Y0A0Nm1
Fbi/bmTL4zeyxfzDYkMSzzb3rVra9r07OMd4zTeLozowODAOBgNVHQ8BAf8EBAMC
AYYwDwYDVR0TAQH/BAUwAwEB/zAVBgNVHREEDjAMhwQAAAAAhwR/AAABMAoGCCqG
SM49BAMCA0kAMEYCIQCDw7CMJ8V2ZvKwPx4uRpb0WFOfDn28mah0FqiCenbGqQIh
AM2I0IByWzc+9vOcoHB1sl8DY2128sSWwTBlMPU8A6yt
-----END CERTIFICATE-----
`
	crtPem = `
-----BEGIN CERTIFICATE-----
MIICmDCCAj+gAwIBAgIIFiYYYP2g1WgwCgYIKoZIzj0EAwIwgawxCzAJBgNVBAYT
AkNOMRAwDgYDVQQIEwdCZWlqaW5nMRkwFwYDVQQHExBIYWlkaWFuIERpc3RyaWN0
MRUwEwYDVQQJEwxCYWlkdSBDYW1wdXMxDzANBgNVBBETBjEwMDA5MzEeMBwGA1UE
ChMVTGludXggRm91bmRhdGlvbiBFZGdlMQ8wDQYDVQQLEwZCQUVUWUwxFzAVBgNV
BAMTDmRlZmF1bHQuMDcyOTAxMB4XDTIwMDcyOTAyMzIwMloXDTQwMDcyNDAyMzIw
Mlowga0xCzAJBgNVBAYTAkNOMRAwDgYDVQQIEwdCZWlqaW5nMRkwFwYDVQQHExBI
YWlkaWFuIERpc3RyaWN0MRUwEwYDVQQJEwxCYWlkdSBDYW1wdXMxDzANBgNVBBET
BjEwMDA5MzEeMBwGA1UEChMVTGludXggRm91bmRhdGlvbiBFZGdlMQ8wDQYDVQQL
EwZCQUVUWUwxGDAWBgNVBAMTD2JhZXR5bC1mdW5jdGlvbjBZMBMGByqGSM49AgEG
CCqGSM49AwEHA0IABH0y7lZWNCo512UgbZFzbZodPk+aO0fX14TXzITqnmYoK7Rk
9rTSprk8lx7JwVxTz6Opv7XKh7yDknpPSSLq7QKjSDBGMA4GA1UdDwEB/wQEAwIF
oDAPBgNVHSUECDAGBgRVHSUAMAwGA1UdEwEB/wQCMAAwFQYDVR0RBA4wDIcEAAAA
AIcEfwAAATAKBggqhkjOPQQDAgNHADBEAiAC3PluuUxcoVnvz8JtaHrQumEJNeo/
Ja9CCrkp24b8rQIgT/+ZbszAFlVO76iI7AtgoJ0cg7hUFjZHVgxh3diCuhY=
-----END CERTIFICATE-----
`
	keyPem = `
-----BEGIN EC PRIVATE KEY-----
MHcCAQEEILraKdvNbV2kwWHbCNecVCvaWJezGthwxTZfMtDCAV4aoAoGCCqGSM49
AwEHoUQDQgAEfTLuVlY0KjnXZSBtkXNtmh0+T5o7R9fXhNfMhOqeZigrtGT2tNKm
uTyXHsnBXFPPo6m/tcqHvIOSek9JIurtAg==
-----END EC PRIVATE KEY-----
`
)

func prepareCert(t *testing.T) string {
	tmpDir, err := ioutil.TempDir("", "init")
	assert.Nil(t, err)
	crt := path.Join(tmpDir, "crt.pem")
	err = ioutil.WriteFile(crt, []byte(crtPem), 0755)
	assert.Nil(t, err)
	ca := path.Join(tmpDir, "ca.pem")
	err = ioutil.WriteFile(ca, []byte(caPem), 0755)
	assert.Nil(t, err)
	key := path.Join(tmpDir, "key.pem")
	err = ioutil.WriteFile(key, []byte(keyPem), 0755)
	assert.Nil(t, err)
	return tmpDir
}

func TestRule(t *testing.T) {
	cli := &Client{}
	clients := map[string]*Client{
		"cli1": cli,
	}

	ruleInfo := RuleInfo{
		Name: "",
		Source: struct {
			QOS   uint32 `yaml:"qos" json:"qos" validate:"min=0, max=1"`
			Topic string `yaml:"topic" json:"topic" validate:"nonzero"`
		}{
			QOS:   1,
			Topic: "t1",
		},
		Target: struct {
			Client string `yaml:"client" json:"client" default:"baetyl-broker"`
		}{
			Client: "cli1",
		},
	}

	dir := prepareCert(t)
	defer os.RemoveAll(dir)
	conf, err := ioutil.TempFile(dir, "conf.yml")
	assert.NoError(t, err)
	ctx := context.NewContext(conf.Name())
	ctx.SystemConfig().Certificate.CA = path.Join(dir, "ca.pem")
	ctx.SystemConfig().Certificate.Cert = path.Join(dir, "crt.pem")
	ctx.SystemConfig().Certificate.Key = path.Join(dir, "key.pem")

	ruler, err := NewRuler(ctx, ruleInfo, clients)
	assert.NoError(t, err)
	time.Sleep(time.Second)
	ruler.Close()
}
