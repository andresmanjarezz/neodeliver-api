package settings

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/graphql-go/graphql"
	"neodeliver.com/engine/rbac"
)

// queries:
type SecuritySettings struct {
	TwoFactorEnabled bool `bson:"two_factor_enabled" json:"two_factor_enabled"`
	// ...
}

func (Query) SecuritySettings(p graphql.ResolveParams, rbac rbac.RBAC) (SecuritySettings, error) {
	// TODO
	return SecuritySettings{}, nil
}

// mutations:

type UpdatePassword struct {
	Old string `bson:",omitempty" validate:"omitempty,lte=255"`
	New string `bson:",omitempty" validate:"omitempty,lte=255"`
}

// UpdatePassword updates the password at Auth0.
func (Mutation) UpdatePassword(p graphql.ResolveParams, rbac rbac.RBAC, args UpdatePassword) (bool, error) {

	auth0 := InitAuth()
	token, err := getManagementToken()
	if err != nil {
		return false, fmt.Errorf("failed to get Auth0 management token: %v", err)
	}

	client := &http.Client{}
	url := fmt.Sprintf("https://%s/api/v2/users/%s", auth0.tenant, rbac.UserID)

	// Prepare request body and marshal into bytes
	body := map[string]interface{}{
		"password":   args.New,
		"connection": auth0.connection,
	}
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return false, fmt.Errorf("failed to marshal JSON body: %v", err)
	}

	// Create HTTP request and headers
	req, err := http.NewRequest(http.MethodPatch, url, strings.NewReader(string(jsonBody)))
	if err != nil {
		return false, fmt.Errorf("failed to create HTTP request: %v", err)
	}

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", token))

	// Do HTTP request
	resp, err := client.Do(req)
	if err != nil {
		return false, fmt.Errorf("failed to make HTTP request: %v", err)
	}
	defer resp.Body.Close()

	// Read the response body
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, fmt.Errorf("failed to read response body: %v", err)
	}

	var response map[string]interface{}
	err = json.Unmarshal(responseBody, &response)
	if err != nil {
		return false, fmt.Errorf("failed to unmarshal JSON response: %v", err)
	}

	// Check the response status code
	if resp.StatusCode != http.StatusOK {
		var message interface{}
		message = "Unable to update password"

		if msg, ok := response["message"]; ok {
			message = msg
		}

		return false, fmt.Errorf("%v", message)
	}

	return true, nil
}
