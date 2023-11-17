package scalars

import (
	"github.com/graphql-go/graphql"
	"github.com/graphql-go/graphql/language/ast"

	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/big"
	"os"
	"strconv"
	"strings"

	log "github.com/inconshreveable/log15"
)

var DecimalScalarType = graphql.NewScalar(graphql.ScalarConfig{
	Name:        "Decimal",
	Description: "Big decimal type.",
	// Serialize serializes `CustomID` to string.
	Serialize: func(value interface{}) interface{} {
		// log.Warn("test", "val", value)
		switch value := value.(type) {
		case Decimal:
			return value.String()
		case *Decimal:
			return value.String()
		default:
			return nil
		}
	},
	// ParseValue parses GraphQL variables from `string` to `CustomID`.
	ParseValue: func(value interface{}) interface{} {
		switch value := value.(type) {
		case string:
			return NewDecimal(value)
		case *string:
			return NewDecimal(*value)
		default:
			return nil
		}
	},
	// ParseLiteral parses GraphQL AST value to `CustomID`.
	ParseLiteral: func(valueAST ast.Value) interface{} {
		switch valueAST := valueAST.(type) {
		case *ast.StringValue:
			return NewDecimal(valueAST.Value)
		default:
			return nil
		}
	},
})

// ----

var pow8Int, _ = big.NewInt(0).SetString("100000000", 10)
var Zero = NewIntDecimal(0)

// 8 decimals number
type Decimal struct {
	i *big.Int
}

// --------------

func FloatToDecimal(f float64) Decimal {
	floor := math.Floor(f)
	i := big.NewInt(int64(floor))
	i = i.Mul(i, pow8Int)
	i = i.Add(i, big.NewInt(int64((f-floor)*100000000)))

	return Decimal{
		i: i,
	}
}

func NewDecimalFromPrecision(precision int) Decimal {
	if precision > 8 {
		precision = 8
	}

	r := int64(math.Pow(float64(10), float64(8-precision)))
	return NewIntDecimal(r)
}

func NewDecimal(v string) Decimal {
	d := Decimal{}
	if err := d.SetString(v); err != nil {
		log.Error("Could not parse decimal", "v", v)
		os.Exit(1)
	}

	return d
}

func NewRawDecimal(v string) Decimal {
	str, ok := big.NewInt(0).SetString(v, 10)
	if !ok {
		log.Error("Could not convert string to decimal", "val", v)
		os.Exit(1)
	}

	return Decimal{
		i: str,
	}
}

func NewBigIntDecimal(i *big.Int) Decimal {
	return Decimal{
		i: i,
	}
}

func NewInt32Decimal(i int) Decimal {
	return NewIntDecimal(int64(i))
}

func NewIntDecimal(i int64) Decimal {
	return Decimal{
		i: big.NewInt(i),
	}
}

// --------------

func (a Decimal) Ref() *Decimal {
	return &a
}

func (a Decimal) Add(b Decimal) Decimal {
	if a.i == nil {
		return b
	} else if b.i == nil {
		return a
	}

	return Decimal{
		i: big.NewInt(0).Add(a.i, b.i),
	}
}

func (a Decimal) Sub(b Decimal) Decimal {
	if a.i == nil {
		if b.i == nil {
			return b
		}

		return b.Neg()
	} else if b.i == nil {
		return a
	}

	return Decimal{
		i: big.NewInt(0).Sub(a.i, b.i),
	}
}

func (a Decimal) Div(b Decimal) Decimal {
	if a.i == nil || b.i == nil {
		return NewIntDecimal(0)
	}

	res := big.NewInt(0).Mul(a.i, pow8Int)
	return Decimal{
		i: big.NewInt(0).Quo(res, b.i),
	}
}

func (a Decimal) RawDiv(b int64) Decimal {
	if a.i == nil || b == 0 {
		return NewIntDecimal(0)
	}

	res := big.NewInt(b)
	res = res.Quo(a.i, res)
	return Decimal{
		i: res,
	}
}

func (a Decimal) Mul(b Decimal) Decimal {
	if a.i == nil || b.i == nil {
		return NewIntDecimal(0)
	}

	res := big.NewInt(0).Mul(a.i, b.i)
	return Decimal{
		i: res.Quo(res, pow8Int),
	}
}

func (a Decimal) Mod(b Decimal) Decimal {
	return Decimal{
		i: big.NewInt(0).Mod(a.i, b.i),
	}
}

func (a Decimal) Pow(b Decimal) Decimal {
	n := a
	for i := int64(0); i < b.Int64(false); i++ {
		n = n.Mul(a)
	}

	for i := b.Int64(false); i < 0; i++ {
		n = n.Div(a)
	}

	return n
}

func (a Decimal) Floor(decimals int) Decimal {
	if decimals >= 8 {
		return a
	}

	b := big.NewInt(100000000)
	for i := 0; i < decimals; i++ {
		b = b.Div(b, big.NewInt(10))
	}

	mod := big.NewInt(0).Mod(a.i, b)
	return Decimal{
		i: big.NewInt(0).Sub(a.i, mod),
	}
}

func (a Decimal) Ceil(decimals int) Decimal {
	if decimals >= 8 {
		return a
	}

	b := big.NewInt(100000000)
	for i := 0; i < decimals; i++ {
		b = b.Div(b, big.NewInt(10))
	}

	mod := big.NewInt(0).Mod(a.i, b)
	res := Decimal{
		i: big.NewInt(0).Sub(a.i, mod),
	}

	if res.Eq(a) {
		return res
	}

	res.i = big.NewInt(0).Add(res.i, b)
	return res
}

func (a Decimal) Round(decimals int) Decimal {
	if decimals >= 8 {
		return a
	}

	b := big.NewInt(100000000)
	for i := 0; i < decimals; i++ {
		b = b.Div(b, big.NewInt(10))
	}

	mod := big.NewInt(0).Mod(a.i, b)
	res := Decimal{
		i: big.NewInt(0).Sub(a.i, mod),
	}

	if mod.Cmp(big.NewInt(0).Div(b, big.NewInt(2))) == 1 {
		res.i = res.i.Add(res.i, b)
	}

	return res
}

func (a Decimal) RoundUpByStep(step Decimal) Decimal {
	n := a.RoundByStep(step)
	if n.Lt(a) {
		return n.Add(step)
	}

	return n
}

func (a Decimal) RoundByStep(step Decimal) Decimal {
	return a.Sub(a.Mod(step))
}

func (a Decimal) Greatest(b Decimal) Decimal {
	if a.Gt(b) {
		return a
	}

	return b
}

func (a Decimal) Least(b Decimal) Decimal {
	if a.Lt(b) {
		return a
	}

	return b
}

func (a Decimal) Cmp(b Decimal) int {
	return a.i.Cmp(b.i)
}

func (a Decimal) Eq(b Decimal) bool {
	if b.i == nil && a.i == nil {
		return true
	} else if b.i == nil || a.i == nil {
		return false
	}

	return a.i.Cmp(b.i) == 0
}

func (a Decimal) Ne(b Decimal) bool {
	return !a.Eq(b)
}

func (a Decimal) Gt(b Decimal) bool {
	if b.i == nil || a.i == nil {
		return false
	}

	return a.i.Cmp(b.i) == 1
}

func (a Decimal) Lt(b Decimal) bool {
	if b.i == nil || a.i == nil {
		return false
	}

	return a.i.Cmp(b.i) == -1
}

func (a Decimal) Gte(b Decimal) bool {
	if b.i == nil || a.i == nil {
		return false
	}

	return a.i.Cmp(b.i) >= 0
}

func (a Decimal) Lte(b Decimal) bool {
	if b.i == nil || a.i == nil {
		return false
	}

	return a.i.Cmp(b.i) <= 0
}

func (a Decimal) Abs() Decimal {
	return Decimal{
		i: big.NewInt(0).Abs(a.i),
	}
}

func (a Decimal) CmpZero() int {
	if a.i == nil {
		return 0
	}

	return a.i.Cmp(big.NewInt(0))
}

func (a Decimal) IsZero() bool {
	if a.i == nil {
		return true
	}

	return a.CmpZero() == 0
}

func (a Decimal) Neg() Decimal {
	return Decimal{
		i: big.NewInt(0).Mul(a.i, big.NewInt(-1)),
	}
}

func (a Decimal) IsNil() bool {
	return a.i == nil
}

func (a Decimal) Highest(b Decimal) Decimal {
	if a.i == nil {
		return b
	} else if b.i == nil {
		return a
	} else if a.Cmp(b) == 1 {
		return a
	}

	return b
}

func (a Decimal) Lowest(b Decimal) Decimal {
	if a.i == nil {
		return b
	} else if b.i == nil {
		return a
	} else if a.Cmp(b) == -1 {
		return a
	}

	return b
}

// ----------------------------------------------------

func (d *Decimal) SetScientificString(v string) error {
	index := strings.Index(strings.ToUpper(v), "E")
	if err := d.SetString(v[:index]); err != nil {
		return err
	}

	n, err := strconv.Atoi(v[index+1:])
	if err != nil {
		return err
	}

	pow := big.NewInt(10)

	if n < 0 {
		for i := n; i < 0; i++ {
			d.i = d.i.Div(d.i, pow)
		}

	} else {
		for i := 0; i < n; i++ {
			d.i = d.i.Mul(d.i, pow)
		}
	}

	return nil
}

func (d *Decimal) SetString(v string) error {
	if strings.Contains(v, "e") || strings.Contains(v, "E") {
		return d.SetScientificString(v)
	}

	initial := v
	if i := strings.Index(v, "."); i == -1 {
		v += "00000000"
	} else {
		if i == 0 {
			v = "0" + v
			i++
		}

		v = v[:i] + (v[i+1:] + "00000000")[:8]
	}

	str, ok := big.NewInt(0).SetString(v, 10)
	if !ok {
		return errors.New("Could not convert string to decimal " + initial)
	}

	d.i = str
	return nil
}

func (a Decimal) String() string {
	if a.i == nil {
		return "0"
	}

	v := a.i.String()
	neg := len(v) > 0 && v[0] == '-'
	if neg {
		v = v[1:]
	}

	for len(v) < 9 {
		v = "0" + v
	}

	l := len(v)
	v = v[:l-8] + "." + v[l-8:]
	v = ShortenDecimalString(v)
	if neg {
		v = "-" + v
	}

	return v
}

func (a Decimal) Float64() float64 {
	f, _ := strconv.ParseFloat(a.String(), 64)
	return f
}

func (a Decimal) Int64(raw bool) int64 {
	if raw {
		return a.i.Int64()
	}

	return int64(a.Float64())
}

func (f *Decimal) Scan(value interface{}) error {
	if value == nil {
		f.i = nil
	} else if v, ok := value.([]uint8); ok {
		dec := NewDecimal(B2S(v))
		f.i = dec.i
	} else if str, ok := value.(string); ok {
		dec := NewDecimal(str)
		f.i = dec.i
	} else {
		return errors.New("Could not assign value to decimal")
	}

	return nil
}

func (f Decimal) Value() (driver.Value, error) {
	if f.i == nil {
		return nil, nil
	}

	return f.String(), nil
}

func (f Decimal) MarshalJSON() ([]byte, error) {
	return []byte("\"" + f.String() + "\""), nil
}

func (f *Decimal) UnmarshalJSON(bs []byte) error {
	if len(bs) == 0 || string(bs) == "\"\"" || string(bs) == "null" {
		f.i = nil
		return nil
	} else if bs[0] == '"' && len(bs) > 2 {
		return f.SetString(string(bs[1 : len(bs)-1]))
	}

	return f.SetString(string(bs))
}

func (d Decimal) Format(f fmt.State, c rune) {
	val := d.String()
	p, _ := f.Precision()

	// 115 = s
	if c == 100 || (c == 102 && p == 0) {
		if i := strings.Index(val, "."); i > 0 {
			val = val[:i]
		}
	} else if c == 102 {
		i := strings.Index(val, ".")
		if i == -1 {
			val += ".0"
			i = len(val) - 2
		}

		for len(val)-i <= p {
			val += "0"
		}

		if len(val)-i > p+1 {
			val = val[:i+p+1]
		}
	}

	f.Write([]byte(val))
}

func (f *Decimal) GobDecode(bs []byte) (err error) {
	if string(bs) == "null" {
		f.i = nil
		return nil
	}

	defer func() {
		// recover from panic if one occured. Set err to nil otherwise.
		if recover() != nil {
			err = errors.New("gob decode failed")
		}
	}()

	return f.i.GobDecode(bs)
}

func (f Decimal) GobEncode() ([]byte, error) {
	return f.i.GobEncode()
}

func (f Decimal) Number() json.Number {
	return json.Number(f.String())
}

func Max(a Decimal, b Decimal) Decimal {
	if a.Gte(b) {
		return a
	} else {
		return b
	}
}

func Min(a Decimal, b Decimal) Decimal {
	if a.Lte(b) {
		return a
	} else {
		return b
	}
}

// ------------------------------------------------

func ShortenDecimalString(res string) string {
	res = strings.TrimRight(res, "0")
	if res == "" {
		res = "0"
	} else if res[len(res)-1] == '.' {
		res = res[:len(res)-1]
	}

	return res
}

func B2S(bs []uint8) string {
	b := make([]byte, len(bs))
	for i, v := range bs {
		b[i] = byte(v)
	}

	return string(b)
}
