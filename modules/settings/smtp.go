package settings

type SMTP struct {
	OrganizationID  string `bson:"_id" json:"organization_id"`
	TlsOnly         bool   `json:"tls_only"`
	AllowSelfSigned bool   `json:"allow_self_signed"`
	Domains         []SMTPDomain
	IPs             []SMTPIp
}

func (s SMTP) Default(organization_id string) SMTP {
	return SMTP{
		OrganizationID:  organization_id,
		TlsOnly:         true,
		AllowSelfSigned: false,
		Domains:         []SMTPDomain{}, // TODO auto genete an internal domain used for testing purposes by customer
	}
}

// ---

type SMTPDomain struct {
	Host      string
	Verified  bool
	TXTRecord string `json:"txt_record"`
	Region    string
	MailsSent int
}

type SMTPIp struct {
	IP        string
	Region    string
	WarmingUp bool
	MailsSent int
}

// TODO add domain name verification
// TODO add users to smtp server
// TODO add dedicated IPs
