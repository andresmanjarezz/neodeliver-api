package settings

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
)

var auth *auth0

type auth0 struct {
	tenant       string
	clientID     string
	clientSecret string
	audience     string
	connection   string
}

// InitAuth initializes Auth0 configurations.
func InitAuth() *auth0 {
	if auth == nil {
		auth = &auth0{
			tenant:       os.Getenv("AUTH0_TENANT"),
			clientID:     os.Getenv("AUTH0_MANAGEMENT_CLIENT_ID"),
			clientSecret: os.Getenv("AUTH0_MANAGEMENT_CLIENT_SECRET"),
			audience:     os.Getenv("AUTH0_MANAGEMENT_AUDIENCE"),
			connection:   os.Getenv("AUTH0_MANAGEMENT_CONNECTION"),
		}
		return auth
	}
	return auth
}

// getManagementToken retrieves the Auth0 management token.
func getManagementToken() (string, error) {

	auth0 := InitAuth()

	// Auth0 token endpoint URL
	tokenURL := fmt.Sprintf("https://%s/oauth/token", auth0.tenant)

	// Prepare request parameters
	params := url.Values{}
	params.Add("grant_type", "client_credentials")
	params.Add("client_id", auth0.clientID)
	params.Add("client_secret", auth0.clientSecret)
	params.Add("audience", auth0.audience)

	// Create a request body with URL-encoded parameters
	reader := strings.NewReader(params.Encode())
	client := &http.Client{}

	// Create an HTTP request
	req, err := http.NewRequest(http.MethodPost, tokenURL, reader)
	if err != nil {
		return "", err
	}
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	// Send the HTTP request
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// Read the response body
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var managementToken map[string]interface{}
	err = json.Unmarshal(responseBody, &managementToken)
	if err != nil {
		return "", err
	}

	// Extract the access token
	token, ok := managementToken["access_token"].(string)
	if !ok {
		return "", errors.New("cannot obtain management token")
	}

	return token, nil
}
