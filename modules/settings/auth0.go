package settings

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	parser "net/url"

	"github.com/golang-jwt/jwt"
	"github.com/joho/godotenv"
	"neodeliver.com/engine/rbac"
)

var auth *auth0

type auth0 struct {
	tenant           string
	clientID         string
	clientSecret     string
	passClientID     string
	passClientSecret string
	audience         string
	connection       string
	token            string
	tokenExp         time.Time
}

// InitAuth initializes Auth0 configurations.
func Auth0() *auth0 {
	if auth == nil {
		auth = &auth0{
			tenant:           os.Getenv("AUTH0_TENANT"),
			token:            os.Getenv("AUTH0_TOKEN"),
			clientID:         os.Getenv("AUTH0_MANAGEMENT_CLIENT_ID"),
			clientSecret:     os.Getenv("AUTH0_MANAGEMENT_CLIENT_SECRET"),
			audience:         os.Getenv("AUTH0_MANAGEMENT_AUDIENCE"),
			connection:       os.Getenv("AUTH0_MANAGEMENT_CONNECTION"),
			passClientID:     os.Getenv("AUTH0_PASSWORD_CLIENT_ID"),
			passClientSecret: os.Getenv("AUTH0_PASSWORD_CLIENT_SECRET"),
		}

		if auth.token != "" {
			if err := auth.updateTokenExpiration(); err != nil {
				fmt.Printf("Failed to update auth0 token expiration: %v\n", err)
			}
		}
	}

	return auth
}

func (a *auth0) getToken(ctx context.Context) (string, error) {
	if a.token != "" && a.tokenExp.After(time.Now()) {
		return a.token, nil
	} else if a.tenant == "" {
		return "", errors.New("auth0 tenant is not set. Please set the AUTH0_TENANT environment variable")
	}

	if a.clientID == "" {
		return "", errors.New("auth0 client id is not set. Please set the AUTH0_MANAGEMENT_CLIENT_ID environment variable")
	} else if a.clientSecret == "" {
		return "", errors.New("auth0 client secret is not set. Please set the AUTH0_MANAGEMENT_CLIENT_SECRET environment variable")
	}

	res := struct {
		Error string
		Token string `json:"access_token"`
	}{}

	if a.audience == "" {
		a.audience = fmt.Sprintf("https://%s/api/v2/", a.tenant)
	}

	_, _, err := a.Request(ctx, "POST", "/oauth/token", nil, map[string]interface{}{
		"grant_type":    "client_credentials",
		"client_id":     a.clientID,
		"client_secret": a.clientSecret,
		"audience":      a.audience,
	}, &res)

	if err != nil {
		return "", err
	} else if res.Token == "" && res.Error != "" {
		return "", errors.New(res.Error)
	} else if res.Token == "" {
		return "", errors.New("could not find auth0 token")
	}

	a.token = res.Token
	if err = a.updateTokenExpiration(); err != nil {
		return res.Token, err
	}

	// save to local env
	if mp, err := godotenv.Read(".env"); err == nil {
		mp["AUTH0_TOKEN"] = res.Token
		godotenv.Write(mp, ".env")
	}
	return res.Token, nil
}

func (a *auth0) updateTokenExpiration() error {
	claims := auth0Claims{}
	parser := jwt.Parser{}

	_, _, err := parser.ParseUnverified(a.token, &claims)
	a.tokenExp = time.Unix(claims.Exp-15, 0)
	return err
}

func (a *auth0) Request(ctx context.Context, method string, url string, headers map[string]string, body interface{}, res interface{}) ([]byte, int, error) {
	if a.tenant == "" {
		return nil, 401, errors.New("auth0 tenant is not set. Please set the AUTH0_TENANT environment variable")
	}

	// Prepare request parameters
	var reader io.Reader
	if body != nil {
		bs, err := json.Marshal(body)
		if err != nil {
			return nil, 0, err
		}

		reader = bytes.NewReader(bs)
	}

	// Create an HTTP request
	if !strings.HasPrefix(url, "https://") {
		url = fmt.Sprintf("https://%s%s", a.tenant, url)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, reader)
	if err != nil {
		return nil, 0, err
	}

	req.Header.Add("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Add(k, v)
	}

	rawURL, err := parser.Parse(url)
	if err != nil {
		return nil, 0, err
	}

	// if the header does not have Authorization key and url is not /oauth/token,  gets the management token
	_, ok := headers["Authorization"]
	if rawURL.Path != "/oauth/token" && !ok {
		token, err := a.getToken(ctx)
		if err != nil {
			return nil, 401, err
		} else if token == "" {
			return nil, 401, errors.New("could ont find auth0 token")
		}

		req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", token))
	}
	// Send the HTTP request
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, http.StatusInternalServerError, err
	}

	defer resp.Body.Close()

	// Read the response body
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, err
	}

	return responseBody, resp.StatusCode, json.Unmarshal(responseBody, res)
}

func (a *auth0) Post(ctx context.Context, url string, headers map[string]string, body interface{}, res interface{}) ([]byte, int, error) {
	return a.Request(ctx, "POST", url, headers, body, res)
}

func (a *auth0) Patch(ctx context.Context, url string, headers map[string]string, body interface{}, res interface{}) ([]byte, int, error) {
	return a.Request(ctx, "PATCH", url, headers, body, res)
}

func (a *auth0) Get(ctx context.Context, url string, headers map[string]string, res interface{}) ([]byte, int, error) {
	return a.Request(ctx, "GET", url, headers, nil, res)
}

func (a *auth0) Delete(ctx context.Context, url string, headers map[string]string, res interface{}) ([]byte, int, error) {
	return a.Request(ctx, "DELETE", url, headers, nil, res)
}

// ----

func (a *auth0) VerifyPassword(ctx context.Context, user_id string, password string) (bool, string, error) {
	if a.passClientID == "" {
		return false, "", errors.New("auth0 password client id is not set. Please set the AUTH0_PASSWORD_CLIENT_ID environment variable")
	} else if a.passClientSecret == "" {
		return false, "", errors.New("auth0 password client secret is not set. Please set the AUTH0_PASSWORD_CLIENT_SECRET environment variable")
	}

	res := struct {
		Error            string
		ErrorDescription string `json:"error_description"`
		MFAToken         string `json:"mfa_token"`
		AccessToken      string `json:"access_token"`
		Email            string
		Identities       []struct {
			UserID     string `json:"user_id"`
			Provider   string `json:"provider"`
			Connection string `json:"connection"`
		}
	}{}

	// find username & connection
	_, _, err := a.Get(ctx, "/api/v2/users/"+user_id, nil, &res)
	if err != nil {
		return false, "", err
	} else if res.Error != "" {
		return false, "", errors.New(res.Error)
	} else if res.Email == "" {
		return false, "", errors.New("could not find user email")
	}

	connection := ""
	for _, identity := range res.Identities {
		if identity.Provider+"|"+identity.UserID == user_id {
			connection = identity.Connection
			break
		}
	}

	if connection == "" {
		return false, "", errors.New("could not find user connection")
	}

	// verify password
	_, _, err = a.Post(ctx, "/oauth/token", nil, map[string]interface{}{
		"grant_type":    "password",
		"client_id":     a.passClientID,
		"client_secret": a.passClientSecret,
		"scope":         "read:sample",
		"connection":    connection,
		"username":      res.Email,
		"password":      password,
	}, &res)

	if err != nil {
		return false, connection, err
	}

	// check the error is mfa_required
	if res.Error != "" && res.Error == "mfa_required" {
		// TODO : Implement the otp verification and get the token
		// unset the error and error description
		// res.Error = ""
		// res.ErrorDescription = ""

		// // verify the MFA and get back the token
		// _, err := a.VerifyMFA(ctx, res.MFAToken, "", "otp", &res)
		// if err != nil {
		// 	return false, connection, err
		// }

		return true, connection, nil
	}

	if res.Error != "" && res.Error != "invalid_grant" {
		return false, connection, errors.New(res.Error)
	}

	return res.AccessToken != "", connection, nil
}

// Verify user mfa
func (a *auth0) VerifyMFA(ctx context.Context, token, verificationCode, method string, res interface{}) (bool, error) {
	var payload = map[string]interface{}{
		"client_id":     a.passClientID,
		"client_secret": a.passClientSecret,
		"mfa_token":     token,
	}

	switch method {
	case "otp":
		payload["grant_type"] = "http://auth0.com/oauth/grant-type/mfa-otp"
		payload["otp"] = verificationCode
	default:
		return false, errors.New("invalid mfa method")
	}

	_, _, err := a.Post(ctx, "/oauth/token", nil, payload, &res)
	if err != nil {
		return false, err
	}

	return true, nil
}

// Enroll user for MFA
func (a *auth0) EnrollMFA(ctx context.Context, types []string, token, method string, res interface{}) (bool, error) {
	headers := map[string]string{
		"Authorization": fmt.Sprintf("Bearer %v", token),
	}
	payload := map[string]interface{}{
		"authenticator_types": types,
	}
	_, _, err := auth.Post(ctx, "/mfa/associate", headers, payload, &res)
	if err != nil {
		return false, err
	}

	return true, nil
}

// Confirm MFA enrollment
func (a *auth0) ConfirmMFAEnrollment(ctx context.Context, token, verificationCode, method string, res interface{}) (bool, error) {
	return a.VerifyMFA(ctx, token, verificationCode, method, res)
}

// Set enrollment authentication method
func (a *auth0) EnrollAuthenticationMethod(ctx context.Context, userID, secret string, res interface{}) (bool, error) {

	payload := map[string]interface{}{
		"type":        "totp",
		"name":        "TOTP",
		"totp_secret": secret, // should be a base32 encoded secret
	}
	url := fmt.Sprintf("/api/v2/users/%v/authentication-methods", userID)
	_, _, err := auth.Post(ctx, url, nil, payload, res)
	if err != nil {
		return false, err
	}
	return true, nil
}

func (a *auth0) LinkAccount(ctx context.Context, rbac rbac.RBAC, args LinkAccount, res interface{}) (bool, error) {

	if args.LinkWith != nil {
		return a.LinkAccountWithJWT(ctx, rbac.Token, *args.LinkWith, rbac.UserID, res)
	}

	payload := map[string]interface{}{}
	if args.Provider != nil && args.UserID != nil {
		payload["provider"] = args.Provider
		payload["user_id"] = args.UserID
	} else {
		return false, errors.New("invalid payload")
	}

	if args.ConnectionID != nil {
		payload["connection_id"] = args.ConnectionID
	}

	url := fmt.Sprintf("/api/v2/users/%s/identities", rbac.UserID)
	_, _, err := auth.Post(ctx, url, nil, payload, res)
	if err != nil {
		return false, err
	}

	return true, nil
}

func (a *auth0) LinkAccountWithJWT(ctx context.Context, primaryToken, secondaryToken, userID string, res interface{}) (bool, error) {
	url := fmt.Sprintf("/api/v2/users/%s/identities", userID)

	_, _, err := auth.Post(ctx, url,
		map[string]string{
			"Authorization": fmt.Sprintf("Bearer %s", primaryToken),
		},
		map[string]interface{}{
			"link_with": secondaryToken,
		},
		res,
	)

	return err == nil, err
}

func (a *auth0) UnlinkAccount(ctx context.Context, provider, userID, secondaryUserID string, res interface{}) (bool, error) {

	url := fmt.Sprintf("/api/v2/users/%s/identities/%s/%s", userID, provider, secondaryUserID)
	_, _, err := auth.Delete(ctx, url, nil, res)
	if err != nil {
		return false, err
	}

	return true, nil
}

// update the user metadata
func (a *auth0) UpdateMetaData(ctx context.Context, userID string, metaData interface{}, res interface{}) (bool, error) {
	url := fmt.Sprintf("/api/v2/users/%v", userID)
	_, _, err := auth.Patch(ctx, url, nil, metaData, res)
	return err == nil, err
}

func (a *auth0) EnableMFA(ctx context.Context, userID string, flag bool, res interface{}) (bool, error) {
	payload := map[string]interface{}{
		"user_metadata": map[string]interface{}{
			"use_mfa": flag,
		},
	}

	return a.UpdateMetaData(ctx, userID, payload, res)
}

type auth0Claims struct {
	Exp int64
}

func (a auth0Claims) Valid() error {
	if time.Unix(a.Exp, 0).Before(time.Now()) {
		return errors.New("auth0 token is expired")
	}

	return nil
}
