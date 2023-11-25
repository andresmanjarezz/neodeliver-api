package graphql

import (
	"reflect"

	pluralize "github.com/gertd/go-pluralize"
	"github.com/graphql-go/graphql"
)

type Builder struct {
	query        graphql.Fields
	mutation     graphql.Fields
	subscription graphql.Fields
	builder      *TypesBuilder
}

func New() *Builder {
	return &Builder{
		builder: NewTypesBuilder(),
	}
}

// ---

func (g *Builder) AddQueryMethods(o interface{}) {
	queryFields := g.builder.ExtractFields(reflect.ValueOf(o), nil)

	if g.query == nil {
		g.query = queryFields
	} else {
		for k, v := range queryFields {
			if _, ok := g.query[k]; ok {
				panic("duplicate query method: " + k)
			}

			g.query[k] = v
		}
	}
}

func (g *Builder) AddSubscriptionMethods(o interface{}) {
	queryFields := g.builder.ExtractFields(reflect.ValueOf(o), nil)

	if g.subscription == nil {
		g.subscription = queryFields
	} else {
		for k, v := range queryFields {
			if _, ok := g.subscription[k]; ok {
				panic("duplicate subscription method: " + k)
			}

			g.subscription[k] = v
		}
	}
}

func (g *Builder) AddMutationMethods(o interface{}) {
	fields := g.builder.ExtractFields(reflect.ValueOf(o), func(p graphql.ResolveParams) bool {
		// admin_id := p.Context.Value("admin_id").(int)
		// return admin_id == 4 || admin_id == 1
		return true
	})

	if g.mutation == nil {
		g.mutation = fields
	} else {
		for k, v := range fields {
			if _, ok := g.mutation[k]; ok {
				panic("duplicate mutation method: " + k)
			}

			g.mutation[k] = v
		}
	}
}

// ---

// auto build mongodb query resolver
func (g *Builder) MongoQuery(o interface{}) *QueryParams {
	// init query map
	if g.query == nil {
		g.query = graphql.Fields{}
	}

	t := reflect.TypeOf(o)
	many := t.Kind() == reflect.Slice
	if many {
		t = t.Elem()
	}

	// get graphql query name
	name := ToSnakeCase(t.Name())
	pluralName := pluralize.NewClient().Plural(name)
	// TODO support item methods when list provided
	if v, ok := o.(interface{ GraphqlName() string }); ok {
		name = v.GraphqlName()
	}

	config := &QueryParams{
		collection: pluralName,
		build:      g.builder,
	}

	field := &graphql.Field{
		Name: name,
		Type: g.builder.Type(reflect.TypeOf(o), false, true),
	}

	if many {
		field.Name = pluralName
		field.Resolve = config.resolveMany(reflect.TypeOf(o))
	} else {
		field.Resolve = config.resolveOne(reflect.TypeOf(o))
	}

	if v, ok := o.(interface{ DeprecationReason() string }); ok {
		field.DeprecationReason = v.DeprecationReason()
	}

	if v, ok := o.(interface{ Description() string }); ok {
		field.DeprecationReason = v.Description()
	}

	config.field = field

	if many {
		name = pluralName

		config.Arg("first", graphql.ArgumentConfig{
			Type:         graphql.Int,
			DefaultValue: 10,
			Description:  "take first n items of list",
		})

		config.Arg("offset", graphql.ArgumentConfig{
			Type:         graphql.Int,
			DefaultValue: 0,
			Description:  "skip n items of list",
		})
	}

	if _, ok := g.query[name]; ok {
		panic("duplicate query method: " + name)
	}

	g.query[name] = field
	return config
}

// ---

func (g *Builder) Build() (graphql.Schema, error) {

	schemaConfig := graphql.SchemaConfig{
		// Directives   []*Directive
		// Extensions   []Extension
		Types: g.builder.GetTypes(),
	}

	if g.query != nil {
		schemaConfig.Query = graphql.NewObject(graphql.ObjectConfig{Name: "RootQuery", Fields: g.query})
	}

	if g.mutation != nil {
		schemaConfig.Mutation = graphql.NewObject(graphql.ObjectConfig{Name: "RootMutation", Fields: g.mutation})
	}

	if g.subscription != nil {
		schemaConfig.Subscription = graphql.NewObject(graphql.ObjectConfig{Name: "RootSubscription", Fields: g.subscription})
	}

	return graphql.NewSchema(schemaConfig)
}
