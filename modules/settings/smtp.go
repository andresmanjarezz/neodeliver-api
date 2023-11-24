package settings

import (
	"context"
	"errors"
	"regexp"
	"strings"

	"github.com/graphql-go/graphql"
	"github.com/segmentio/ksuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"neodeliver.com/engine/db"
	"neodeliver.com/engine/rbac"
)

var regions = map[string]bool{
	"eu": true,
	"us": true,
}

type SMTP struct {
	OrganizationID  string       `bson:"_id" json:"organization_id"`
	TlsOnly         bool         `json:"tls_only" bson:"tls_only"`
	AllowSelfSigned bool         `bson:"allow_self_signed"`
	Domains         []SMTPDomain `bson:"domains"`
	IPs             []SMTPIp     `json:"ips" bson:"ips"`
	UpdateToken     string       `json:"-" bson:"update_token"` // used to assert data has not changed during update functions
}

func (s SMTP) Default(organization_id string) SMTP {
	return SMTP{
		OrganizationID:  organization_id,
		TlsOnly:         true,
		AllowSelfSigned: false,
		Domains:         []SMTPDomain{}, // TODO auto genete an internal domain used for testing purposes by customer
	}
}

func (s *SMTP) Save(ctx context.Context) error {
	oldToken := s.UpdateToken
	s.UpdateToken = ksuid.New().String()

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

type SMTPDomain struct {
	Host      string
	Verified  bool
	Region    *string
	MailsSent int
}

type NewDomain struct {
	Host   string
	Region *string
}

func (Mutation) AddDomain(p graphql.ResolveParams, rbac rbac.RBAC, args NewDomain) (SMTP, error) {
	s := SMTP{}

	// verify region
	if args.Region != nil {
		*args.Region = strings.ToLower(*args.Region)
		if !regions[*args.Region] {
			return s, errors.New("invalid_region")
		}
	}

	// verify host
	args.Host = strings.ToLower(args.Host)
	r := regexp.MustCompile(`^(([a-zA-Z0-9]|[a-zA-Z0-9][a-zA-Z0-9\-]*[a-zA-Z0-9])\.)*([A-Za-z0-9]|[A-Za-z0-9][A-Za-z0-9\-]*[A-Za-z0-9])$`)
	if !r.Match([]byte(args.Host)) {
		return s, errors.New("invalid_host")
	}

	// find organization smtp settings
	s, err := GetSMTPSettings(p.Context, rbac.OrganizationID)
	if err != nil {
		return s, err
	}

	// check if domain already exists
	for _, d := range s.Domains {
		if d.Host == args.Host {
			return s, errors.New("domain_already_exists")
		}
	}

	// register domain
	s.Domains = append(s.Domains, SMTPDomain{
		Host:      args.Host,
		Region:    args.Region,
		Verified:  false,
		MailsSent: 0,
	})

	// save smtp settings
	err = s.Save(p.Context)
	return s, err
}

// ---

type SMTPIp struct {
	IP        string
	Region    string
	WarmingUp bool
	MailsSent int
}

// TODO add domain name verification
// TODO add users to smtp server
// TODO add dedicated IPs
