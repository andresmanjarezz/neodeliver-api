package graphql

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"runtime"
	"runtime/debug"
	"strings"
	"time"

	"github.com/getsentry/sentry-go"
	validate "github.com/go-playground/validator/v10"
	"github.com/graphql-go/graphql"
	"neodeliver.com/engine/rbac"
)

// Building graphql queries
// -------------------------------------------------------------------------------------

// Parse graphql method response
type argumentsCreatorFn = func(graphql.ResolveParams, []reflect.Value) ([]reflect.Value, error)

func resultify(res []reflect.Value) (interface{}, error) {
	if len(res) == 0 {
		return nil, nil
	}

	v1 := res[0].Interface()
	if x, ok := v1.(error); ok {
		return nil, x
	} else if len(res) == 1 {
		return v1, nil
	}

	v2, _ := res[1].Interface().(error)
	return v1, v2
}

func assignValue(field reflect.StructField, dest reflect.Value, data reflect.Value) error {
	if dest.Kind() == reflect.Ptr {
		val := reflect.New(field.Type.Elem())
		if dest.Type().Elem().Kind() == reflect.Struct {
			if bs, err := json.Marshal(data.Interface()); err != nil {
				return err
			} else if err = json.Unmarshal(bs, dest.Addr().Interface()); err != nil {
				return err
			}
			return assignValue(field, val.Elem(), data)
		}
		val.Elem().Set(data)
		dest.Set(val)
		return nil
	} else if dest.Kind() == reflect.Struct {
		if bs, err := json.Marshal(data.Interface()); err != nil {
			return err
		} else if err = json.Unmarshal(bs, dest.Addr().Interface()); err != nil {
			return err
		}
	} else if dest.Kind() == reflect.Slice {
		valueSlice := reflect.MakeSlice(dest.Type(), data.Len(), 1024)
		for i := 0; i < data.Len(); i++ {
			err := assignValue(reflect.StructField{
				Type: data.Index(i).Elem().Type(),
			}, valueSlice.Index(i), data.Index(i).Elem())

			if err != nil {
				return err
			}
		}

		dest.Set(valueSlice)
	} else {
		dest.Set(data)
	}

	return nil
}

// Querry arguments
func (t *TypesBuilder) graphqlArguments(in reflect.Type, superCreate argumentsCreatorFn) (graphql.FieldConfigArgument, argumentsCreatorFn) {
	names := []string{}

	createArguments := func(p graphql.ResolveParams, v []reflect.Value) ([]reflect.Value, error) {
		var err error
		if superCreate != nil {
			v, err = superCreate(p, v)
			if err != nil {
				return v, err
			}
		}

		val := reflect.New(in)
		for i, name := range names {
			if v, ok := p.Args[name]; ok {
				field := val.Elem().Field(i)

				err = assignValue(val.Elem().Type().Field(i), field, reflect.ValueOf(v))
				if err != nil {
					return nil, err
				}
			}
		}

		err = validate.New().StructCtx(p.Context, val.Elem().Interface())
		return append(v, val.Elem()), err
	}

	args := graphql.FieldConfigArgument{}

	// Add graphql arguments
	for f := 0; f < in.NumField(); f++ {
		arg := in.Field(f)
		if arg.Anonymous {
			panic("anonymous fields are not supported on args")
		}

		name := ToSnakeCase(arg.Name)

		if n := arg.Tag.Get("graphql"); n != "" {
			name = n
		}

		names = append(names, name)
		kind := t.Type(arg.Type, true, false)

		args[name] = &graphql.ArgumentConfig{
			Type: kind,
		}
	}

	return args, createArguments
}

// Transform graphql query method
func (t *TypesBuilder) toGraphqlResolver(v reflect.Value, locked func(graphql.ResolveParams) bool) (graphql.FieldResolveFn, graphql.FieldConfigArgument) {
	args := graphql.FieldConfigArgument{}
	var createArguments argumentsCreatorFn

	mixin := func(cb func(p graphql.ResolveParams, v []reflect.Value) ([]reflect.Value, error)) {
		if createArguments == nil {
			createArguments = cb
		} else {
			super := createArguments
			createArguments = func(p graphql.ResolveParams, v []reflect.Value) ([]reflect.Value, error) {
				v, err := super(p, v)
				if err != nil {
					return nil, err
				}

				return cb(p, v)
			}
		}
	}

	// Parse query argumentss
	for i := 0; i < v.Type().NumIn(); i++ {
		in := v.Type().In(i)
		if n := in.Name(); n == "ResolveParams" {
			mixin(func(p graphql.ResolveParams, v []reflect.Value) ([]reflect.Value, error) {
				return append(v, reflect.ValueOf(p)), nil
			})

			continue
		} else if n == "RBAC" {
			mixin(func(p graphql.ResolveParams, v []reflect.Value) ([]reflect.Value, error) {
				rbac, err := rbac.FromContext(p.Context)
				return append(v, reflect.ValueOf(rbac)), err
			})

			continue
		}

		args, createArguments = t.graphqlArguments(in, createArguments)
	}

	// Create callback
	if createArguments == nil {
		createArguments = func(p graphql.ResolveParams, v []reflect.Value) ([]reflect.Value, error) { return v, nil }
	}

	return func(p graphql.ResolveParams) (interface{}, error) {
		if locked != nil && !locked(p) {
			return nil, errors.New("action not permitted")
		}
		// fields := p.Info.FieldASTs[0].SelectionSet.Selections[1].(*ast.Field).SelectionSet.Selections
		// colls := sql.ExtractFields(reflect.TypeOf(User{}), fields, sql.ExtractConfig{})
		// log15.Info(strings.Join(colls, ", "))
		// log15.Info(sql.ToQuery("users", colls, ""))

		defer func() {
			if r := recover(); r != nil {
				fmt.Println("Recovered GraphQL error", r)
				fmt.Println(string(debug.Stack()))
				sentry.CurrentHub().Recover(r)
				sentry.Flush(time.Second)
				panic(r)
			}
		}()

		args, err := createArguments(p, []reflect.Value{})
		if err != nil {
			return nil, err
		}

		res := v.Call(args)
		return resultify(res)
	}, args
}

func (t *TypesBuilder) Method(fn interface{}, locked func(graphql.ResolveParams) bool) (string, *graphql.Field) {
	v := reflect.ValueOf(fn)
	name := runtime.FuncForPC(v.Pointer()).Name()
	name = ToSnakeCase(name[strings.LastIndex(name, ".")+1:])

	// Create resolver and get arguments
	resolve, args := t.toGraphqlResolver(v, locked)
	field := &graphql.Field{
		Resolve: func(p graphql.ResolveParams) (interface{}, error) {
			res, err := resolve(p)
			if err != nil {
				return nil, err
			}

			res = resultToGraphqlMap(res)
			bs, _ := json.MarshalIndent(res, "", "  ")
			fmt.Println(string(bs))
			return res, nil
		},
		Args: args,
	}

	// Add output type
	if v.Type().NumOut() > 0 {
		out := v.Type().Out(0)
		if out.Name() != "error" {
			field.Type = t.Type(out, false, false)
		}
	}

	// Return method as graphql field
	return name, field
}

func (t *TypesBuilder) ExtractFields(kind reflect.Value, locked func(graphql.ResolveParams) bool) graphql.Fields {
	// kind := reflect.TypeOf(q)
	res := graphql.Fields{}

	// Add fields
	// for i:=0; i<kind.NumField(); i++ {
	// 	field := kind.Field(i)
	// 	n := ToSnakeCase(field.Name)
	// 	fields[n] = &graphql.Field{
	// 		Type: t.Type(field.Type),
	// 	}
	// }

	for i := 0; i < kind.NumMethod(); i++ {
		method := kind.Method(i)
		// Decomment to debug graphql builder
		// log15.Info("TypesBuilder.ExtractFields", "i", i, "method", method)
		_, field := t.Method(method.Interface(), locked)
		name := ToSnakeCase(kind.Type().Method(i).Name)
		res[name] = field
	}

	return res
}

// ---

func resultToGraphqlMap(i interface{}) interface{} {
	if i == nil {
		return i
	}

	t := reflect.TypeOf(i)
	if t.Kind() == reflect.Pointer || t.Kind() == reflect.UnsafePointer {
		t = t.Elem()
		i = reflect.ValueOf(i).Elem().Interface()
	}

	// convert lists
	if t.Kind() == reflect.Slice {
		res := []interface{}{}
		val := reflect.ValueOf(i)

		for i := 0; i < val.Len(); i++ {
			res = append(res, resultToGraphqlMap(val.Index(i).Interface()))
		}

		return res
	}

	// convert objects
	if t.Kind() == reflect.Struct && t.Name() != "Time" {
		res := map[string]interface{}{}
		resultToGraphqlMapToMap(i, res)
		return res
	}

	return i
}

func resultToGraphqlMapToMap(i interface{}, res map[string]interface{}) {
	val := reflect.ValueOf(i)
	t := reflect.TypeOf(i)

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if !field.IsExported() {
			continue
		}

		name := field.Name
		spl := strings.Split(field.Tag.Get("json"), ",")

		if n := field.Tag.Get("graphql"); n != "" {
			name = n
		} else if n := spl[0]; n != "" {
			name = n
		} else {
			name = ToSnakeCase(name)
		}

		f := val.Field(i)
		if field.Anonymous {
			if !f.IsZero() {
				resultToGraphqlMapToMap(f.Interface(), res)
			}

			continue
		}

		if f.IsZero() {
			res[name] = nil
		} else {
			res[name] = resultToGraphqlMap(f.Interface())
		}
	}
}
