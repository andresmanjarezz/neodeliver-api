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
func Auth() *auth0 {
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

	auth0 := Auth()

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

// updat the user password
func updateUserPassword(token string, userID string, args UpdatePassword) error {

	auth := Auth()
	client := &http.Client{}
	url := fmt.Sprintf("https://%s/api/v2/users/%s", auth.tenant, userID)

	// Prepare request body and marshal into bytes
	body := map[string]interface{}{
		"password":   args.New,
		"connection": auth.connection,
	}
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("failed to marshal JSON body: %v", err)
	}

	// Create HTTP request and headers
	req, err := http.NewRequest(http.MethodPatch, url, strings.NewReader(string(jsonBody)))
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %v", err)
	}

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", token))

	// Do HTTP request
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to make HTTP request: %v", err)
	}
	defer resp.Body.Close()

	// Read the response body
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %v", err)
	}

	var response map[string]interface{}
	err = json.Unmarshal(responseBody, &response)
	if err != nil {
		return fmt.Errorf("failed to unmarshal JSON response: %v", err)
	}

	// Check the response status code
	if resp.StatusCode != http.StatusOK {
		var message interface{}
		message = "Unable to update password"

		if msg, ok := response["message"]; ok {
			message = msg
		}

		return fmt.Errorf("%v", message)
	}
	return nil
}
