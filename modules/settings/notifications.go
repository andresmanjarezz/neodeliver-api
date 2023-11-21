package settings

import (
	"github.com/graphql-go/graphql"
	"neodeliver.com/engine/rbac"
)

type UserNotifications struct {
	Tips       bool
	Updates    bool
	Promotions bool
	Security   bool
	Reports    bool
}

func (Mutation) EditNotificationPreferences(p graphql.ResolveParams, rbac rbac.RBAC, args UserNotifications) (UserNotifications, error) {
	// TODO update contact prefernces within our own mailing list (our organization id is accesible from env avriable NEODELIVER_ORGANIZATION_ID)

	return args, nil
}
