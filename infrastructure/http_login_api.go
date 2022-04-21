package infrastructure

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"gitlab.loc/sdk-login/login-sdk-go/interfaces"
	"gitlab.loc/sdk-login/login-sdk-go/model"
)

const (
	Timeout              = 3 * time.Second
	ValidateTokenAPIPATH = "/api/token/validate"
	ProjectsPath         = "/api/projects/"
)

type HTTPLoginAPI struct {
	Client  *http.Client
	baseURL string
}

func NewHttpLoginAPI(baseUrl string, ignoreSslErrors bool) interfaces.LoginAPI {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: ignoreSslErrors},
	}

	return &HTTPLoginAPI{&http.Client{Timeout: Timeout, Transport: tr}, baseUrl}
}

func (api *HTTPLoginAPI) makeRequest(ctx context.Context, method string, url string, body []byte) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	response, err := api.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed make request: %w", err)
	}

	return response, nil
}

func (api *HTTPLoginAPI) GetProjectKeysForLoginProject(ctx context.Context, projectID string) ([]model.ProjectPublicKey, error) {
	response, err := api.makeRequest(ctx, http.MethodGet, fmt.Sprintf("%s%s%s/keys", api.baseURL, ProjectsPath, projectID), nil)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	respBody, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("failed read response: %w", err)
	}

	var keysResp []model.ProjectPublicKey

	if err = json.Unmarshal(respBody, &keysResp); err != nil {
		return nil, fmt.Errorf("failed unmarshal data: %w", err)
	}

	return keysResp, nil
}

func (api *HTTPLoginAPI) ValidateHS256Token(ctx context.Context, token string) error {
	values := map[string]string{"token": token}
	data, err := json.Marshal(values)
	if err != nil {
		return fmt.Errorf("failed marshal data %w", err)
	}

	response, err := api.makeRequest(ctx, http.MethodPost, fmt.Sprintf("%s%s", api.baseURL, ValidateTokenAPIPATH), data)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusNoContent {
		return fmt.Errorf("http request error: %d", response.StatusCode)
	}

	return nil
}
