package main

import (
	"encoding/json"

	"github.com/baetyl/baetyl-go/v2/errors"
	"github.com/baetyl/baetyl-go/v2/http"
	v1 "github.com/baetyl/baetyl-go/v2/spec/v1"
)

const (
	StsUrl = "/agent/sts"
)

func GetSts(cli *http.Client) (*v1.STSResponse, error) {
	var err error
	req := &v1.STSRequest{
		STSType: "minio",
	}
	reqBytes, err := json.Marshal(req)
	if err != nil {
		return nil, errors.Trace(err)
	}
	body, err := cli.PostJSON(StsUrl, reqBytes)
	if err != nil {
		return nil, errors.Trace(err)
	}
	var stsInfo v1.STSResponse
	if err = json.Unmarshal(body, &stsInfo); err != nil {
		return nil, errors.Trace(err)
	}
	return &stsInfo, nil
}
