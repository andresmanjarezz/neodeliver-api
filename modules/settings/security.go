package settings

import (
	"errors"
	"fmt"

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
	Old string `bson:",omitempty" validate:"gt=0,lte=255"`
	New string `bson:",omitempty" validate:"gte=8,lte=255"`
	// TODO : Enable this if the user needs to pass the verification token to change the password
	// VerificationCode string `bson:",omitempty"`
}

// UpdatePassword updates the password at Auth0.
func (Mutation) UpdatePassword(p graphql.ResolveParams, rbac rbac.RBAC, args UpdatePassword) (bool, error) {
	auth := Auth0()

	// verify current password
	ok, connection, err := auth.VerifyPassword(p.Context, rbac.UserID, args.Old)
	if err != nil {
		fmt.Println(err)
		// TODO log to sentry
		return false, errors.New("internal_error")
	} else if !ok {
		return false, fmt.Errorf("invalid_password")
	}

	// update password
	res := struct {
		StatusCode int    `json:"statusCode"`
		Message    string `json:"message"`
		Error      string
	}{}

	url := fmt.Sprintf("/api/v2/users/%s", rbac.UserID)
	bs, status, err := auth.Patch(p.Context, url, nil, map[string]interface{}{
		"password":   args.New,
		"connection": connection,
	}, &res)

	// verify response
	if err != nil {
		fmt.Println(err)
		// TODO log to sentry
		return false, errors.New("internal_error")
	} else if res.StatusCode == 400 {
		return false, fmt.Errorf(res.Message)
	} else if status != 200 {
		fmt.Println(string(bs))
		return false, fmt.Errorf("failed to update password")
	}

	return true, err
}

type Auth0Method struct {
	ID        string
	Type      string
	Confirmed bool
}

func (Query) ListMFA(p graphql.ResolveParams, rbac rbac.RBAC) ([]Auth0Method, error) {

	auth := Auth0()

	var res []Auth0Method
	url := fmt.Sprintf("/api/v2/users/%s/authentication-methods", rbac.UserID)
	_, _, err := auth.Get(p.Context, url, nil, &res)
	if err != nil {
		return nil, err
	}

	return res, nil
}

type EnrollMFA struct {
	Type string `bson:",omitempty"`
}

type ConfirmMFAEnroll struct {
	CurrentPassword  string `bson:",omitempty"`
	Type             string `bson:",omitempty"`
	VerificationCode string `bson:",omitempty"`
}
type MFAResponse struct {
	Error   string `json:"error"`
	Message string `json:"error_description"`

	AuthenticatorType string `json:"authenticator_type"`
	BarcodeURI        string `json:"barcode_uri"`
	Secret            string `json:"secret"`
}

type Identities struct {
	UserID     string `json:"user_id"`
	Provider   string `json:"provider"`
	Connection string `json:"connection"`
}
type LoginResponse struct {
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description"`
	AccessToken      string `json:"access_token"`
	IDToken          string `json:"id_token"`
	Scope            string `json:"scope"`
	ExpiresIn        int    `json:"expires_in"`
	TokenType        string `json:"token_type"`
}

// EnrollMFA
// Enroll user for new MFA
func (Mutation) EnrollMFA(p graphql.ResolveParams, rbac rbac.RBAC, args EnrollMFA) (MFAResponse, error) {

	auth := Auth0()

	res := MFAResponse{}
	_, err := auth.EnrollMFA(p.Context, []string{args.Type}, rbac.Token, "otp", &res)
	if err != nil {
		return res, err
	}
	if res.Error != "" {
		return res, errors.New(res.Message)
	}

	return res, nil
}

// ConfirmMFA
// Confirm the user mfa
func (Mutation) ConfirmMFA(p graphql.ResolveParams, rbac rbac.RBAC, args ConfirmMFAEnroll) (LoginResponse, error) {
	auth := Auth0()
	res := LoginResponse{}

	// verify current password
	ok, _, err := auth.VerifyPassword(p.Context, rbac.UserID, args.CurrentPassword)
	if err != nil {
		fmt.Println(err)
		// TODO log to sentry
		return res, errors.New("internal_error")
	} else if !ok {
		return res, fmt.Errorf("invalid_password")
	}

	// check the current password
	_, err = auth.ConfirmMFAEnrollment(p.Context, rbac.Token, args.VerificationCode, args.Type, &res)
	if err != nil {
		return res, err
	}
	if res.Error != "" {
		return res, errors.New(res.ErrorDescription)
	}

	return res, nil
}

// Enroll New authentication Method for user
// As this will be add the authentication method for user by the management id
type EnrollAuthentication struct {
	CurrentPassword string
	Secret          string
}

type EnrollAuthenticationResponse struct {
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description"`

	ID        string `json:"id,omitempty"`
	Type      string
	Name      string
	CreatedAt string
}

func (Mutation) EnrollAuthenticationMethod(p graphql.ResolveParams, rbac rbac.RBAC, args EnrollAuthentication) (EnrollAuthenticationResponse, error) {
	auth := Auth0()
	res := EnrollAuthenticationResponse{}

	// verify current password
	ok, _, err := auth.VerifyPassword(p.Context, rbac.UserID, args.CurrentPassword)
	if err != nil {
		fmt.Println(err)
		// TODO log to sentry
		return res, errors.New("internal_error")
	} else if !ok {
		return res, fmt.Errorf("invalid_password")
	}

	// check the current password
	_, err = auth.EnrollAuthenticationMethod(p.Context, rbac.UserID, args.Secret, &res)
	if err != nil {
		return res, err
	}
	if res.Error != "" {
		return res, errors.New(res.ErrorDescription)
	}

	return res, nil

}
