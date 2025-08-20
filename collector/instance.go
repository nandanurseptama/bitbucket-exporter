package collector

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/nandanurseptama/bitbucket-exporter/config"
)

type instance struct {
	*http.Client
	*config.AuthConfig
	baseUrl string
}

func newInstance(authConfig *config.AuthConfig) *instance {
	return &instance{
		Client:     http.DefaultClient,
		AuthConfig: authConfig,
		baseUrl:    "https://api.bitbucket.org/2.0",
	}
}
func (i *instance) GetDefaultHeaders() http.Header {
	header := make(http.Header)
	header.Add("Accept", "application/json")
	switch i.AuthConfig.Type {
	case "basic":
		auth := i.AuthConfig.Basic.Username + ":" + i.AuthConfig.Basic.Password
		basicAuth := "Basic " + base64.StdEncoding.EncodeToString([]byte(auth))
		header.Add("Authorization", basicAuth)
		return header
	case "oauth2":
		// TODO : implementation
		return header
	}
	return header
}

func (i *instance) GET(
	ctx context.Context,
	endpoint string,
	params map[string]string,
	respBodyDest any,
) error {
	uri := strings.Join([]string{i.baseUrl, endpoint}, "/")
	req, err := http.NewRequestWithContext(ctx, "GET", uri, nil)

	if err != nil {
		return fmt.Errorf("instance err : %v", err)
	}
	req.Header = i.GetDefaultHeaders()

	q := req.URL.Query() // url.Values

	for key, value := range params {
		q.Set(key, value)
	}

	req.URL.RawQuery = q.Encode()
	fmt.Println(req.URL.String())

	res, err := i.Do(req)

	if err != nil {
		return fmt.Errorf("instance err : %v", err)
	}

	defer res.Body.Close()

	bodyRes, err := io.ReadAll(res.Body)
	if err != nil {
		return fmt.Errorf("instance err : %v", err)
	}

	err = json.Unmarshal(bodyRes, respBodyDest)

	if err != nil {
		return fmt.Errorf("unmarshal response body err : %v", err)
	}

	return nil
}
