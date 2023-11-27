package settings

import (
	"neodeliver.com/engine/graphql"
	"neodeliver.com/engine/rbac"
)

// register graphql queries
func Init(s *graphql.Builder) {
	// query user
	// s.MongoQuery(User{}).Where(func(r rbac.RBAC) map[string]interface{} {
	// 	return map[string]interface{}{
	// 		"_id": r.UserID,
	// 	}
	// })

	// get team members list
	s.MongoQuery([]TeamMember{}).Where(func(r rbac.RBAC) map[string]interface{} {
		return map[string]interface{}{
			"organization_id": r.OrganizationID,
			"deleted_at":      nil,
		}
	})

	// query contact_settings
	s.MongoQuery(ContactSettings{}).Where(func(r rbac.RBAC) map[string]interface{} {
		return map[string]interface{}{
			"_id": r.OrganizationID,
		}
	})

	// query smtp
	s.MongoQuery(SMTP{}).Where(func(r rbac.RBAC) map[string]interface{} {
		return map[string]interface{}{
			"_id": r.OrganizationID,
		}
	})

	s.AddQueryMethods(Query{})
	s.AddMutationMethods(Mutation{})
}

type Mutation struct{}
type Query struct{}
