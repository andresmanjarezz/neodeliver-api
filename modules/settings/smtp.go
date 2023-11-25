package settings

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"net"
	"regexp"
	"strings"
	"time"

	"github.com/graphql-go/graphql"
	"github.com/inconshreveable/log15"
	"github.com/segmentio/ksuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"neodeliver.com/engine/db"
	"neodeliver.com/engine/rbac"
)

var nonAlphanumericRegex = regexp.MustCompile(`[^a-zA-Z0-9 ]+`)

var regions = map[string]bool{
	"eu": true,
	"us": true,
}

type SMTP struct {
	OrganizationID  string    `bson:"_id" json:"organization_id"`
	TlsOnly         bool      `json:"tls_only" bson:"tls_only"`
	AllowSelfSigned bool      `bson:"allow_self_signed"`
	IPs             []SMTPIp  `json:"ips" bson:"ips"`
	UpdateToken     string    `json:"-" bson:"update_token"` // used to assert data has not changed during update functions
	UpdatedAt       time.Time `json:"updated_at" bson:"updated_at"`
}

func (s SMTP) Default(organization_id string) SMTP {
	return SMTP{
		OrganizationID:  organization_id,
		TlsOnly:         true,
		AllowSelfSigned: false,
	}
}

func (s *SMTP) Save(ctx context.Context) error {
	oldToken := s.UpdateToken
	s.UpdateToken = ksuid.New().String()
	s.UpdatedAt = time.Now()

	if oldToken == "" {
		return db.Create(ctx, s)
	}

	return db.Update(ctx, s, bson.M{"_id": s.OrganizationID, "update_token": oldToken}, s)
}

func GetSMTPSettings(ctx context.Context, organization_id string) (SMTP, error) {
	s := SMTP{}
	err := db.Find(ctx, &s, bson.M{"_id": organization_id})

	if err == mongo.ErrNoDocuments {
		s = s.Default(organization_id)
		err = nil
	}

	return s, err
}

// ---

type SMTPDomainRecord struct {
	Type     string
	Host     string
	Value    string
	Verified bool
}

func (s *SMTPDomainRecord) Verify(ctx context.Context) error {
	res := net.Resolver{
		PreferGo: true,
	}

	s.Verified = false
	if s.Type != "TXT" {
		return errors.New("invalid_record_type")
	}

	lst, err := res.LookupTXT(ctx, s.Host)
	if err != nil && !strings.Contains(err.Error(), "no such host") {
		return err
	} else if err != nil {
		return nil
	}

	for _, v := range lst {
		if v == s.Value {
			s.Verified = true
			return nil
		}
	}

	// TODO support advanced dmarc & SPF checks

	return nil
}

// ---

type SMTPDomain struct {
	ID             string `bson:"_id" graphql:"-"`
	OrganizationID string `bson:"organization_id"`
	Host           string
	Verified       bool
	Region         *string
	MailsSent      int `bson:"mails_sent"`
	Records        []SMTPDomainRecord
	CreatedAt      time.Time `bson:"created_at"`
	UpdatedAt      time.Time `bson:"updated_at"`
	UpdateToken    string    `json:"-" bson:"update_token"` // used to assert data has not changed during update functions
}

func (s *SMTPDomain) Save(ctx context.Context) error {
	oldToken := s.UpdateToken
	s.UpdateToken = ksuid.New().String()
	s.UpdatedAt = time.Now()

	if oldToken == "" {
		return db.Create(ctx, s)
	}

	return db.UpdateOne(ctx, s, bson.M{"_id": s.ID, "update_token": oldToken}, s)
}

func (s *SMTPDomain) GenerateID() string {
	h := sha256.New()
	h.Write([]byte(s.OrganizationID))
	h.Write([]byte(s.Host))
	s.ID = "dom_" + nonAlphanumericRegex.ReplaceAllString(base64.StdEncoding.EncodeToString(h.Sum(nil)), "")
	return s.ID
}

func (s *SMTPDomain) Verify(ctx context.Context) (e error) {
	modified := false
	verifiedRecords := 0

	defer func() {
		if modified {
			s.Verified = verifiedRecords == len(s.Records)

			// save domain
			if err := s.Save(ctx); err != nil && e == nil {
				e = err
			} else if err != nil {
				log15.Error("failed to save smtp domain", "err", err)
			}
		}
	}()

	// verify each record
	for i, record := range s.Records {
		before := record.Verified
		if err := record.Verify(ctx); err != nil {
			return err
		}

		if record.Verified != before {
			modified = true
			s.Records[i] = record
		}

		if record.Verified {
			verifiedRecords++
		}
	}

	return nil
}

// ---

type NewDomain struct {
	Host   string
	Region *string
}

func (Mutation) AddDomain(p graphql.ResolveParams, rbac rbac.RBAC, args NewDomain) (SMTPDomain, error) {
	// verify region
	if args.Region != nil {
		*args.Region = strings.ToLower(*args.Region)
		if !regions[*args.Region] {
			return SMTPDomain{}, errors.New("invalid_region")
		}
	}

	// verify host
	args.Host = strings.ToLower(args.Host)
	r := regexp.MustCompile(`^(([a-zA-Z0-9]|[a-zA-Z0-9][a-zA-Z0-9\-]*[a-zA-Z0-9])\.)*([A-Za-z0-9]|[A-Za-z0-9][A-Za-z0-9\-]*[A-Za-z0-9])$`)
	if !r.Match([]byte(args.Host)) {
		return SMTPDomain{}, errors.New("invalid_host")
	}

	// create domain item (id is generated from org id and host to avoid duplicates)
	dom := SMTPDomain{
		OrganizationID: rbac.OrganizationID,
		Host:           args.Host,
		Region:         args.Region,
		Verified:       false,
		MailsSent:      0,
		CreatedAt:      time.Now(),
		Records: []SMTPDomainRecord{
			{
				Type:  "TXT",
				Host:  "_dmarc." + args.Host,
				Value: "v=DMARC1;  p=none; rua=mailto:442f636fce044ef3998100e0361bbdcd@dmarc-reports.cloudflare.net", // TODO auto generate
			},
			{
				Type:  "TXT",
				Host:  "mail._domainkey." + args.Host,
				Value: "v=DKIM1; k=rsa; p=...", // TODO auto generate
			},
			{
				Type:  "TXT",
				Host:  args.Host,
				Value: "v=spf1 include:eu.neodeliver.io ~all", // TODO auto generate
			},
		},
	}

	dom.GenerateID()
	err := dom.Save(p.Context)
	if mongo.IsDuplicateKeyError(err) {
		err = db.Find(p.Context, &dom, bson.M{"_id": dom.ID})
	}

	return dom, err
}

func (Mutation) VerifyDomain(p graphql.ResolveParams, rbac rbac.RBAC, args struct{ Host string }) (SMTPDomain, error) {
	log15.Debug("verifying domain", "host", args.Host)
	start := time.Now()

	dom := SMTPDomain{
		OrganizationID: rbac.OrganizationID,
		Host:           args.Host,
	}

	if err := db.Find(p.Context, &dom, bson.M{"_id": dom.GenerateID()}); err != nil {
		return SMTPDomain{}, err
	}

	err := dom.Verify(p.Context)
	log15.Debug("finished domain verification", "host", args.Host, "time", time.Since(start))

	return dom, err
}

// ---

type SMTPIp struct {
	IP        string
	Region    string
	WarmingUp bool
	MailsSent int
}

// TODO add users to smtp server
// TODO add dedicated IPs
