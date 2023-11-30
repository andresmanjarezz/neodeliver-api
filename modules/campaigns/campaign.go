package campaigns

import (
	"errors"
	"reflect"
	"time"

	"github.com/graphql-go/graphql"
	"github.com/inconshreveable/log15"
	"github.com/segmentio/ksuid"
	"neodeliver.com/engine/db"
	ggraphql "neodeliver.com/engine/graphql"
	"neodeliver.com/engine/rbac"
	"neodeliver.com/utils"
)

// TODO we should allow each lang to be present only once
// TODO we should validate lang exists
// TODO we should validate at least 1 lang is present
// TODO we should allow up to max 50 recipients (0=allowed), which can be a mix of context ids, tags, segments (we differentiate it with id prefix)
// TODO we should validate recipients exists
// TODO we should add stats (sent, delivered, opened, clicked, unsubscribed, bounced, complained, failed)

type Campaign struct {
	ID             string     `bson:"_id" json:"id"`
	OrganizationID string     `bson:"organization_id" json:"organization_id"`
	Draft          bool       `bson:"draft" json:"draft"`
	CreatedAt      time.Time  `bson:"created_at" json:"created_at"`
	UpdatedAt      time.Time  `bson:"updated_at" json:"updated_at"`
	UpdateToken    string     `bson:"update_token" json:"update_token"`
	NextSchedule   *time.Time `bson:"next_schedule" json:"next_schedule"` // updated by scheduler to perform fast searches on next scheduled campaigns
	CampaignType   string     `bson:"campaign_type" json:"campaign_type"`
	CampaignData   `bson:",inline" json:",inline"`
}

func (c *Campaign) UpdateComputedFields() {
	if c.CampaignData.SMS != nil {
		c.CampaignType = "sms"
	} else if c.CampaignData.Email != nil {
		c.CampaignType = "email"
	} else {
		c.CampaignType = "notification"
	}

	c.NextSchedule = nil
	if c.Scheduler != nil {
		// TODO update next schedule
	}
}

// ----------------------------------------

// data updatable by user
type CampaignData struct {
	Name          string                `bson:"name" json:"name"`
	Recipients    []string              `bson:"recipients" json:"recipients"`
	Scheduler     *CampaignScheduler    `bson:"scheduled" json:"scheduled"`
	Transactional bool                  `bson:"transactional" json:"transactional"`
	Email         *Email                `bson:"email,omitempty" json:"email,omitempty"`
	SMS           *SMSCampaign          `bson:"sms,omitempty" json:"sms,omitempty"`
	Notification  *NotificationCampaign `bson:"notification,omitempty" json:"notification,omitempty"`
}

func (c *CampaignData) Validate() error {
	if countNotNil(c.Email, c.SMS, c.Notification) != 1 {
		log15.Debug("campaign validation", "email", c.Email, "sms", c.SMS, "notification", c.Notification, "count", countNotNil(c.Email, c.SMS, c.Notification))
		return errors.New("only one campaign type can be created at a time") // todo: change to utils....
	}

	// TODO validate at least 1 lang is present
	// TODO validate sub fields formats

	return nil
}

type CampaignScheduler struct {
	AutoOptimizeTime bool   `bson:"auto_optimize_time" json:"auto_optimize_time"`
	UseUserTimezone  bool   `bson:"use_user_timezone" json:"use_user_timezone"`
	RRule            string `bson:"rrule" json:"rrule"`
}

// ----------------------------------------
// email settings

type Email struct {
	SenderName    string      `bson:"sender_name,omitempty" json:"sender_name"`
	SenderEmail   string      `bson:"sender_email,omitempty" json:"sender_email"`
	ClickTracking *bool       `bson:"click_tracking,omitempty" json:"click_tracking"`
	OpenTracking  *bool       `bson:"open_tracking,omitempty" json:"open_tracking"`
	Languages     []EmailLang `bson:"languages,omitempty" json:"languages"`
}

type EmailLang struct {
	Lang        string   `bson:"lang" json:"lang"`
	Subject     string   `bson:"subject" json:"subject"`
	Preheader   string   `bson:"preheader" json:"preheader"`
	Attachments []string `bson:"attachments,omitempty" json:"attachments"`
}

// ----------------------------------------
// sms settings

type SMSCampaign struct {
	Languages []SMSLang `bson:"languages,omitempty" json:"languages"`
}

type SMSLang struct {
	Text string `bson:"text" json:"text"`
}

// ----------------------------------------
// notification settings

type NotificationCampaign struct {
	Languages     []NotificationLang `bson:"languages,omitempty" json:"languages"`
	Platform      string             `bson:"platform,omitempty" json:"platform"`
	OpenURL       *string            `bson:"open_url,omitempty" json:"open_url,omitempty"`
	ClickTracking *bool              `bson:"click_tracking,omitempty" json:"click_tracking,omitempty"`
	OpenTracking  *bool              `bson:"open_tracking,omitempty" json:"open_tracking,omitempty"`
}

type NotificationLang struct {
	Title string   `bson:"title" json:"title"`
	Text  string   `bson:"text" json:"text"`
	Media []string `bson:"media,omitempty" json:"media,omitempty"`
}

// ------------------------------------------------------------------------------------------------------------------------
// ------------------------------------------------------------------------------------------------------------------------
// mutations

func (Mutation) CreateCampaign(p graphql.ResolveParams, rbac rbac.RBAC, args CampaignData) (Campaign, error) {
	e := Campaign{
		ID:             "cmp_" + ksuid.New().String(),
		OrganizationID: rbac.OrganizationID,
		Draft:          true,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
		CampaignData:   args,
		UpdateToken:    ksuid.New().String(),
	}

	if err := e.CampaignData.Validate(); err != nil {
		return e, err
	}

	e.UpdateComputedFields()
	err := db.Create(p.Context, &e)
	return e, err
}

// ---

type CampaignEdit struct {
	ID   string       `json:"id"`
	Data CampaignData `json:"data"` // todo support inline
}

func (Mutation) UpdateCampaign(p graphql.ResolveParams, rbac rbac.RBAC, args CampaignEdit) (Campaign, error) {
	if err := args.Data.Validate(); err != nil {
		return Campaign{}, err
	}

	// find current campaign details
	current := Campaign{}
	err := db.Find(p.Context, &current, map[string]string{
		"_id":             args.ID,
		"organization_id": rbac.OrganizationID,
	})

	if err != nil {
		// todo log to sentry
		return Campaign{}, errors.New(utils.MessageCampaignCannotFindError)
	}

	// validate format
	oldFormat := current.CampaignType
	oldToken := current.UpdateToken
	current.CampaignData = args.Data
	current.UpdatedAt = time.Now()

	if err := current.CampaignData.Validate(); err != nil {
		return current, err
	} else if oldFormat != current.CampaignType {
		return current, errors.New("campaign_type cannot be changed") // TODO change to utils
	}

	current.UpdateComputedFields()
	current.UpdateToken = ksuid.New().String()

	// save new data
	m, err := db.UpdateOne(p.Context, &current, map[string]string{
		"_id":             args.ID,
		"organization_id": rbac.OrganizationID,
		"update_token":    oldToken,
	}, current)

	if err != nil {
		// todo log to sentry
		log15.Error("error updating campaign", "err", err)
		return current, errors.New(utils.MessageDefaultError)
	} else if m.MatchedCount == 0 {
		return current, errors.New("campaign changed during update, please try again")
	}

	return current, nil
}

func (Mutation) DeleteCampaign(p graphql.ResolveParams, rbac rbac.RBAC, filter ggraphql.ByID) (bool, error) {
	err := db.Delete(p.Context, &Campaign{}, map[string]string{
		"_id":             filter.ID,
		"organization_id": rbac.OrganizationID,
	})

	if err != nil {
		// TODO log to sentry
		return false, errors.New(utils.MessageCampaignCannotDeleteError)
	}

	return true, nil
}

// --

func countNotNil(a ...interface{}) int {
	count := 0
	for _, c := range a {
		if c == nil || (reflect.ValueOf(c).Kind() == reflect.Ptr && reflect.ValueOf(c).IsNil()) {
			continue
		}

		count++
	}

	return count
}
