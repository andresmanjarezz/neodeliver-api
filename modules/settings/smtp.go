package settings

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"regexp"
	"sort"
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
		s = SMTP{
			OrganizationID:  organization_id,
			TlsOnly:         true,
			AllowSelfSigned: false,
		}

		err = nil
	}

	return s, err
}

// ---

type SMTPDomainRecord struct {
	Type       string
	Host       string
	Value      string
	AltValue   string `bson:"alt_value,omitempty"` // if host has already an existing record & we are extending it
	Verified   bool
	VerifiedAt time.Time `bson:"verified_at" graphql:"-"`
}

func (s *SMTPDomainRecord) Verify(ctx context.Context) error {
	res := net.Resolver{
		PreferGo: true,
	}

	s.Verified = false
	s.VerifiedAt = time.Now()
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

		if strings.HasPrefix(v, "v=spf1") && strings.HasPrefix(s.Value, "v=spf1") {
			return s.VerifySPF(v)
		} else if strings.HasPrefix(v, "v=DMARC1; ") && strings.HasPrefix(s.Value, "v=DMARC1; ") {
			// TODO add advanced DMARC verification
			s.AltValue = v
			s.Verified = true
			s.VerifiedAt = time.Now()
		}
	}

	return nil
}

func (s *SMTPDomainRecord) VerifySPF(match string) error {
	// extract needed includes
	needed := map[string]bool{}
	for _, item := range strings.Split(s.Value, " ")[1:] {
		if strings.HasPrefix(item, "include:") {
			needed[item] = true
		}
	}

	// parse current host record & construct new record
	n := []string{"v=spf1"}
	hasAll := false

	addMissing := func() {
		lst := []string{}
		for k := range needed {
			lst = append(lst, k)
		}

		sort.Strings(lst)
		n = append(n, lst...)
	}

	for _, item := range strings.Split(match, " ")[1:] {
		if item == "" {
			continue
		}

		op := item[:1]
		if op != "+" && op != "-" && op != "~" && op != "?" {
			op = ""
		} else {
			item = item[1:]
		}

		if _, ok := needed[item]; ok {
			if op != "" && op != "+" && op != "?" {
				op = ""
			}

			delete(needed, item)
			n = append(n, op+item)
		} else if item == "all" {
			hasAll = true
			addMissing()
			n = append(n, op+"all")
			break
		} else {
			n = append(n, op+item)
		}
	}

	if len(needed) > 0 && !hasAll {
		addMissing()
		n = append(n, "~all")
	}

	s.AltValue = strings.Join(n, " ")
	s.Verified = len(needed) == 0

	fmt.Println(match)
	bs, _ := json.MarshalIndent(needed, "", "  ")
	println(string(bs))

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
	DKIMPrivateKey []byte    `graphql:"-" json:"-" bson:"dkim_private_key"`
	CreatedAt      time.Time `bson:"created_at"`
	UpdatedAt      time.Time `bson:"updated_at"`
	VerifiedAt     time.Time `bson:"verified_at"`
	UpdateToken    string    `json:"-" bson:"update_token"` // used to assert data has not changed during update functions
}

func (s *SMTPDomain) Save(ctx context.Context) error {
	oldToken := s.UpdateToken
	s.UpdateToken = ksuid.New().String()
	s.UpdatedAt = time.Now()

	if oldToken == "" {
		return db.Create(ctx, s)
	}

	_, err := db.UpdateOne(ctx, s, bson.M{"_id": s.ID, "update_token": oldToken}, s)
	return err
}

func (s *SMTPDomain) GenerateID() string {
	h := sha256.New()
	h.Write([]byte(s.OrganizationID))
	h.Write([]byte(s.Host))
	s.ID = "dom_" + nonAlphanumericRegex.ReplaceAllString(base64.StdEncoding.EncodeToString(h.Sum(nil)), "")
	return s.ID
}

func (s *SMTPDomain) Verify(ctx context.Context) (e error) {
	verifiedRecords := 0

	defer func() {
		s.Verified = verifiedRecords == len(s.Records)
		s.VerifiedAt = time.Now()

		// save domain
		if err := s.Save(ctx); err != nil && e == nil {
			e = err
		} else if err != nil {
			log15.Error("failed to save smtp domain", "err", err)
		}
	}()

	// verify each record
	for i, record := range s.Records {
		if record.Verified && record.VerifiedAt.After(time.Now().Add(-time.Minute*15)) {
			verifiedRecords++
			continue
		}

		if err := record.Verify(ctx); err != nil {
			return err
		}

		s.Records[i] = record
		if record.Verified {
			verifiedRecords++
		}
	}

	return nil
}

func generateRSAKeyAndDKIMRecord(length int) (*rsa.PrivateKey, string, error) {
	// Generate RSA private key
	privateKey, err := rsa.GenerateKey(rand.Reader, length)
	if err != nil {
		return nil, "", err
	}

	// Encode the public key to DKIM format
	pubKeyBytes, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	if err != nil {
		return nil, "", err
	}

	// Generate DKIM record
	dkimRecord := fmt.Sprintf(
		"v=DKIM1; k=rsa; p=%s",
		base64.StdEncoding.EncodeToString(pubKeyBytes),
	)

	return privateKey, dkimRecord, nil
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

	// generate dkim key
	key, record, err := generateRSAKeyAndDKIMRecord(2048)
	if err != nil {
		return SMTPDomain{}, err
	}

	// create domain item (id is generated from org id and host to avoid duplicates)
	dom := SMTPDomain{
		OrganizationID: rbac.OrganizationID,
		Host:           args.Host,
		Region:         args.Region,
		Verified:       false,
		MailsSent:      0,
		CreatedAt:      time.Now(),
		DKIMPrivateKey: x509.MarshalPKCS1PrivateKey(key),
		Records: []SMTPDomainRecord{
			{
				Type:  "TXT",
				Host:  "_dmarc." + args.Host,
				Value: "v=DMARC1; p=none; fo=1; rua=mailto:admin@" + args.Host,
			},
			{
				Type:  "TXT",
				Host:  "neo._domainkey." + args.Host,
				Value: record,
			},
			{
				Type:  "TXT",
				Host:  args.Host,
				Value: "v=spf1 include:neodeliver.com ~all",
			},
		},
	}

	dom.GenerateID()

	// save domain
	err = dom.Save(p.Context)
	if mongo.IsDuplicateKeyError(err) {
		err = db.Find(p.Context, &dom, bson.M{"_id": dom.ID})
	} else if err == nil {
		err = dom.Verify(p.Context)
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
