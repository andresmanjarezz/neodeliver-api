package graphql

import (
	"reflect"

	"github.com/graphql-go/graphql"
	"neodeliver.com/engine/db"
	"neodeliver.com/engine/graphql/scalars"
)

func ToSnakeCase(str string) string {
	return db.ToSnakeCase(str)
}

// Graphql type builder
// -------------------------------------------------------------------------------------

type TypesBuilder struct {
	types map[string]graphql.Type
}

func NewTypesBuilder() *TypesBuilder {
	return &TypesBuilder{
		types: map[string]graphql.Type{},
	}
}

func (t *TypesBuilder) GetTypes() []graphql.Type {
	res := []graphql.Type{}
	for _, r := range t.types {
		res = append(res, r)
	}

	return res
}

func FieldsFactory(kind reflect.Type, fieldHandler func(string, reflect.StructField)) {
	for i := 0; i < kind.NumField(); i++ {
		field := kind.Field(i)

		name := field.Tag.Get("graphql")
		if name == "-" {
			continue
		} else if name == "" {
			name = ToSnakeCase(field.Name)
		}

		// handle embeded fields
		if field.Anonymous {
			FieldsFactory(field.Type, fieldHandler)
			continue
		}

		fieldHandler(name, field)
		continue
	}
}

func (t *TypesBuilder) InternalType(name string, kind reflect.Type, inputType bool) graphql.Type {
	key := "out_" + name
	if inputType {
		key = "in_" + name
	}

	if v, ok := t.types[key]; ok {
		return v
	}

	if kind.Kind() == reflect.Ptr {
		kind = kind.Elem()
	}

	if inputType {
		fields := graphql.InputObjectConfigFieldMap{}
		FieldsFactory(kind, func(name string, field reflect.StructField) {
			fields[name] = &graphql.InputObjectFieldConfig{
				Type: t.Type(field.Type, inputType, false),
			}
		})

		t.types[key] = graphql.NewInputObject(graphql.InputObjectConfig{
			Name:   "In" + name,
			Fields: fields,
		})
	} else {
		fields := graphql.Fields{}
		FieldsFactory(kind, func(name string, field reflect.StructField) {
			fields[name] = &graphql.Field{
				Type: t.Type(field.Type, inputType, false),
			}
		})

		t.types[key] = graphql.NewObject(graphql.ObjectConfig{
			Name:   name,
			Fields: fields,
		})
	}

	return t.types[key]
}

func (t *TypesBuilder) Type(out reflect.Type, inputType bool, forceNullable bool) graphql.Output {
	ptr := out.Kind() == reflect.Ptr

	if _, ok := out.MethodByName("GraphqlType"); ok {
		if ptr {
			out = out.Elem()
		}

		obj := reflect.New(out).Interface()
		res := obj.(interface{ GraphqlType() graphql.Output })
		return res.GraphqlType()
	}

	name := out.Name()
	if out.Kind() == reflect.Slice {
		kind := t.Type(out.Elem(), inputType, false)
		return graphql.NewList(kind)
	}

	if ptr {
		name = out.Elem().Name()
	}

	// Uncomment to debug graphql builder
	// log15.Info("TypesBuilder.Type", "name", name, "out.Kind()", out.Kind())

	var kind graphql.Output
	switch name {
	case "string":
		kind = graphql.String
	case "int", "int64", "int32":
		kind = graphql.Int
	case "float", "float64", "float32":
		kind = graphql.Float
	case "bool":
		kind = graphql.Boolean
	case "Time":
		kind = graphql.DateTime
	case "Decimal":
		kind = scalars.DecimalScalarType
	case "JSON":
		kind = scalars.JSON
	default:
		switch out.Kind() {
		case reflect.Bool:
			kind = graphql.Boolean
		case reflect.Int:
			kind = graphql.Int
		case reflect.Int8:
			kind = graphql.Int
		case reflect.Int16:
			kind = graphql.Int
		case reflect.Int32:
			kind = graphql.Int
		case reflect.Int64:
			kind = graphql.Int
		case reflect.Uint:
			kind = graphql.Int
		case reflect.Uint8:
			kind = graphql.Int
		case reflect.Uint16:
			kind = graphql.Int
		case reflect.Uint32:
			kind = graphql.Int
		case reflect.Uint64:
			kind = graphql.Int
		case reflect.Float32:
			kind = graphql.Float
		case reflect.Float64:
			kind = graphql.Float
		case reflect.String:
			kind = graphql.String
		default:
			kind = t.InternalType(name, out, inputType)
		}
	}

	if !ptr && !forceNullable {
		return graphql.NewNonNull(kind)
	} else {
		return kind
	}
}
