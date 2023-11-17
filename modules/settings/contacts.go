package settings

import (
	"github.com/graphql-go/graphql"
	"neodeliver.com/engine/rbac"
)

type ContactSettings struct {
	OrganizationID string `bson:"_id" json:"organization_id"`
	Tracking       TrackingSettings
	Email          ContactEmailSettings
	SMS            ContactSMSSettings
	// CommunicationCategories []map[string]string
}

func (ContactSettings) Default(org string) ContactSettings {
	return ContactSettings{
		OrganizationID: org,
		Tracking: TrackingSettings{
			ClickTracking: true,
			OpenTracking:  true,
		},
		Email: ContactEmailSettings{
			BlacklistMode:   false,
			RestrictionList: []string{},
			UnsubscribeLink: false,
		},
		SMS: ContactSMSSettings{
			UnsubscribeLink: false,
		},
	}
}

// ------------------ Sub structs ------------------

type TrackingSettings struct {
	ClickTracking   bool                     `json:"click_tracking"`
	OpenTracking    bool                     `json:"open_tracking"`
	GoogleAnalytics *GoogleAnalyticsSettings `json:"google_analytics"`
}

type ContactEmailSettings struct {
	GoogleAnalytics *GoogleAnalyticsSettings `json:"google_analytics"`
	BlacklistMode   bool                     `json:"blacklist_mode"`
	RestrictionList []string                 `json:"restriction_list"`
	UnsubscribeLink bool                     `json:"unsubscribe_link"`
}

type ContactSMSSettings struct {
	UnsubscribeLink bool `json:"unsubscribe_link"`
}

type GoogleAnalyticsSettings struct {
	Source  string
	Medium  string
	Term    string
	Content string
}

// ------------------ Email restrictions ------------------

type AddRestrictedEmailParams struct {
	Email string
}

func (Mutation) AddRestrictedEmail(p graphql.ResolveParams, rbac rbac.RBAC, args AddRestrictedEmailParams) (ContactSettings, error) {
	// TODO validate email format
	// TODO append email to restriction list

	return (ContactSettings{}).Default(rbac.OrganizationID), nil
}
