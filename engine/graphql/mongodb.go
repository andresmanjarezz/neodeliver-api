package graphql

import (
	"reflect"

	"github.com/graphql-go/graphql"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
	"neodeliver.com/engine/db"
	"neodeliver.com/engine/rbac"
)

type QueryParams struct {
	build      *TypesBuilder
	field      *graphql.Field
	collection string
	where      graphql.FieldResolveFn
	staticArgs graphql.FieldConfigArgument
	whereArgs  graphql.FieldConfigArgument
	// Order
}

func (q *QueryParams) resolveOne(t reflect.Type) graphql.FieldResolveFn {
	getDefault, hasDefault := t.MethodByName("Default")

	return func(p graphql.ResolveParams) (interface{}, error) {
		client := db.Client()
		coll := client.Collection(q.collection)

		// add where clause
		var filter interface{} = bson.D{}
		if q.where != nil {
			var err error
			filter, err = q.where(p)
			if err != nil {
				return nil, err
			}
		}

		// find doc
		res := coll.FindOne(p.Context, filter)
		if err := res.Err(); err != nil {
			if err.Error() == "mongo: no documents in result" {
				if hasDefault {
					r, _ := rbac.FromContext(p.Context)

					// call default function
					res := getDefault.Func.Call([]reflect.Value{
						reflect.New(t).Elem(),
						reflect.ValueOf(r.OrganizationID),
					})

					return res[0].Interface(), nil
				}

				return nil, nil
			}

			return nil, err
		}

		result := reflect.New(t).Interface()
		err := res.Decode(result)
		return result, err
	}
}

func (q *QueryParams) resolveMany(t reflect.Type) graphql.FieldResolveFn {
	return func(p graphql.ResolveParams) (interface{}, error) {
		client := db.Client()
		coll := client.Collection(q.collection)

		// add where clause
		var filter interface{} = bson.D{}
		if q.where != nil {
			var err error
			filter, err = q.where(p)
			if err != nil {
				return nil, err
			}
		}

		first := int64(p.Args["first"].(int))
		skip := int64(p.Args["offset"].(int))
		// fmt.Println("first", first)

		// find doc
		cur, err := coll.Find(p.Context, filter, &options.FindOptions{
			Limit: &first,
			Skip:  &skip,
		})

		if err != nil {
			return nil, err
		}

		result := reflect.New(t).Interface()
		err = cur.All(p.Context, result)

		return result, err
	}
}

func (q *QueryParams) Arg(name string, kind graphql.ArgumentConfig) {
	if q.staticArgs == nil {
		q.staticArgs = graphql.FieldConfigArgument{}
	}

	q.staticArgs[name] = &kind
	q.field.Args = q.mergedArgs()
}

func (q *QueryParams) Where(v interface{}) *QueryParams {
	resolver, args := q.build.toGraphqlResolver(reflect.ValueOf(v), nil)
	q.whereArgs = args
	q.field.Args = q.mergedArgs()
	q.where = resolver
	return q
}

func (q *QueryParams) mergedArgs() graphql.FieldConfigArgument {
	args := q.whereArgs
	if args == nil {
		return q.staticArgs
	}

	if q.staticArgs != nil {
		for k, v := range q.staticArgs {
			args[k] = v
		}
	}

	return nil
}

// execute function after resolve
func (q *QueryParams) After(fn func(graphql.ResolveParams, interface{}) (interface{}, error)) *QueryParams {
	resolve := q.field.Resolve
	q.field.Resolve = func(p graphql.ResolveParams) (interface{}, error) {
		result, err := resolve(p)
		if err != nil {
			return nil, err
		}

		return fn(p, result)
	}

	return q
}

// ---

type ByID struct {
	ID string
}
