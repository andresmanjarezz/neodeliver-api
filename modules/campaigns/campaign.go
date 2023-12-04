package campaigns

import (
	"errors"
	"reflect"
	"time"
	"strings"

	"github.com/graphql-go/graphql"
	"github.com/inconshreveable/log15"
	"github.com/segmentio/ksuid"
	"neodeliver.com/engine/db"
	ggraphql "neodeliver.com/engine/graphql"
	"neodeliver.com/engine/rbac"
	"neodeliver.com/utils"
	"neodeliver.com/modules/contacts"
)

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
	CampaignStats  CampaignStats	`bson:"campaign_stats" json:"campaign_stats"`
}

func (c *Campaign) UpdateComputedFields() {
	if c.CampaignData.SMS != nil {
		c.CampaignType = "sms"
	} else if c.CampaignData.Email != nil {
		c.CampaignType = "email"
	} else {
		c.CampaignType = "notification"
	}

	if c.Scheduler != nil {
		if c.Scheduler.AutoOptimizeTime == true {

		} else {
			deliverTime := time.Date(c.Scheduler.BeginDate.Year(), c.Scheduler.BeginDate.Month(), c.Scheduler.BeginDate.Day(), c.Scheduler.DeliveryTime.Hour, c.Scheduler.DeliveryTime.Minute, 0, 0, time.UTC)
			if deliverTime.Before(time.Now()) {
				deliverTime = deliverTime.Add(24 * time.Hour)
			}
			c.NextSchedule = &deliverTime
		}
	} else {
		deliverTime := time.Now()
		c.NextSchedule = &deliverTime
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

type CampaignStats struct {
	CampaignSent		int		`bson:"campaign_sent" json:"campaign_sent"`
	CampaignDelivered	int		`bson:"campaign_delivered" json:"campaign_delivered"`
	CampaignOpened		int		`bson:"campaign_opened" json:"campaign_opened"`
	CampaignClicked		int		`bson:"campaign_clicked" json:"campaign_clicked"`
	CampaignUnsubed		int		`bson:"campaign_unsubed" json:"campaign_unsubed"`
	CampaignBounced		int		`bson:"campaign_bounced" json:"campaign_bounced"`
	CampaignComplained	int		`bson:"campaign_complained" json:"campaign_complained"`
	CampaignFailed		int		`bson:"campaign_failed" json:"campaign_failed"`
}

func (c *CampaignData) Validate(p graphql.ResolveParams, rbac rbac.RBAC) error {
	if countNotNil(c.Email, c.SMS, c.Notification) != 1 {
		log15.Debug("campaign validation", "email", c.Email, "sms", c.SMS, "notification", c.Notification, "count", countNotNil(c.Email, c.SMS, c.Notification))
		return errors.New(utils.MessageCampaignMustBeOneTypeError)
	}

	if !(c.Email != nil && len(c.Email.Languages) != 0 || c.SMS != nil && len(c.SMS.Languages) != 0 || c.Notification != nil && len(c.Notification.Languages) != 0) {
		return errors.New(utils.MessageCampaignNoLangProvidedError)
	}

	// check number of recipients
	if len(c.Recipients) >= 50 {
		return errors.New(utils.MessageCampaignRecipientExceedLimitError)
	}

	// convert segments into contacts 
	for _, recipient := range c.Recipients {
		if strings.HasPrefix(recipient, "ctc_") {
			err := db.Find(p.Context, &contacts.Contact{}, map[string]string{
				"_id": recipient,
			})
			if err != nil {
				return errors.New(utils.MessageCampaignInvalidRecipientError)
			}
		} else if strings.HasPrefix(recipient, "sgt_") {
			err := db.Find(p.Context, &contacts.Segment{}, map[string]string{
				"_id": recipient,
			})
			if err != nil {
				return errors.New(utils.MessageCampaignInvalidRecipientError)
			}
		} else if strings.HasPrefix(recipient, "tag_") {

		} else {
			return errors.New(utils.MessageCampaignInvalidRecipientError)
		}
	}
	
	// check scheduler
	if c.Scheduler != nil && !c.Scheduler.EndDate.After(*c.Scheduler.BeginDate) {
		return errors.New(utils.MessageCampaignInvalidScheduleError)
	}
	return nil
}

type TTime struct {
	Hour		int		`json:"hour"`
	Minute		int		`json:"minute"`
}

type CampaignScheduler struct {
	AutoOptimizeTime bool   `bson:"auto_optimize_time" json:"auto_optimize_time"`
	UseUserTimezone  bool   `bson:"use_user_timezone" json:"use_user_timezone"`
	RRule            string `bson:"rrule" json:"rrule"`
	HoursOfPeriod	 int	`bson:"hours_of_period" json:"hours_of_period"`
	DeliveryTime	 TTime	`bson:"delivery_time" json:"delivery_time"`
	BeginDate		 *time.Time	`bson:"begin_date" json:"begin_date"`
	EndDate			 *time.Time	`bson:"end_date" json:"end_date"`
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
	Lang string `bson:"lang" json:"lang"`
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
	Lang  string   `bson:"lang" json:"lang"`
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

	if err := e.CampaignData.Validate(p, rbac); err != nil {
		return e, err
	}

	e.UpdateComputedFields()
	err := db.Create(p.Context, &e)
	return Campaign{}, err
}

// ---

type CampaignEdit struct {
	ID   string       `json:"id"`
	Data	CampaignData	`json:",inline"`
}

func (Mutation) UpdateCampaign(p graphql.ResolveParams, rbac rbac.RBAC, args CampaignEdit) (Campaign, error) {
	if err := args.Data.Validate(p, rbac); err != nil {
		return Campaign{}, err
	}

	// find current campaign details
	current := Campaign{}
	err := db.Find(p.Context, &current, map[string]string{
		"_id":             args.ID,
		"organization_id": rbac.OrganizationID,
	})

	if err != nil {
		utils.LogErrorToSentry(err)
		return Campaign{}, errors.New(utils.MessageCampaignCannotFindError)
	}

	// validate format
	oldFormat := current.CampaignType
	oldToken := current.UpdateToken
	current.CampaignData = args.Data
	current.UpdatedAt = time.Now()

	if err := current.CampaignData.Validate(p, rbac); err != nil {
		return current, err
	} else if oldFormat != current.CampaignType {
		return current, errors.New(utils.MessageCampaignCannotChangeTypeError)
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
		utils.LogErrorToSentry(err)
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
		utils.LogErrorToSentry(err)
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

func GetContactsBySegmentQuery(p graphql.ResolveParams, rbac rbac.RBAC, id string) ([]contacts.Contact, error) {
	s := contacts.Segment{}
	err := db.Find(p.Context, &s, map[string]string{
		"_id": id,
		"organization_id": rbac.OrganizationID,
	})
	if err != nil {
		return []contacts.Contact{}, errors.New(utils.MessageSegmentCannotFindError)
	}

	bsonObj, err := utils.ConvertQueryToBSON(*s.Filters)
	if err != nil {
		return []contacts.Contact{}, err
	}

	c := contacts.Contact{}
	contacts_by_segment := []contacts.Contact{}
	err = db.FindAll(p.Context, &c, &contacts_by_segment, bsonObj)
	if err != nil {
		return []contacts.Contact{}, errors.New(utils.MessageDefaultError)
	}
	
	return contacts_by_segment, err
}
