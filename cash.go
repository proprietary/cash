package cash

/*
 * cash—a realistic money type for Go(lang)
 * © 2016 zelcon@zelcon.org
 *
 * The API is extremely similar to "math/big" in the standard library.
 * Most methods do not return allocated memory. This is intentional to
 * reduce unnecessary clock cycles and memory leaks because this is
 * the type of API that you would call A LOT.
 *
 *
 */

import (
	"bytes"
	"database/sql/driver"
	"errors"
	"math/big"
	"strconv"
	"strings"
	"unicode/utf8"
)

type Cash struct {
	Amt        int64
	FracDigits int
	Rational   *big.Rat // nil unless needed
	Currency   rune
	Decimal    rune
	Thousands  rune
}

var MinorUnit = []int64{1, 10, 100, 1000, 10000, 100000, 1000000, 10000000, 100000000, 1000000000, 10000000000}

// presets
var (
	USD = Cash{
		Currency:   '$',
		FracDigits: 2,
		Decimal:    '.',
		Thousands:  ',',
		Rational:   nil,
	}

	EUR = Cash{
		Currency:   '€',
		FracDigits: 2,
		Decimal:    '.',
		Thousands:  ',',
		Rational:   nil,
	}

	BTC = Cash{
		Currency:   '฿',
		FracDigits: 8,
		Decimal:    '.',
		Thousands:  ',',
		Rational:   nil,
	}
)

func New(src Cash) *Cash {
	ret := src
	return &ret
}

// convenience factory for $USD values
func NewUSD() *Cash {
	ret := USD
	return &ret
}

// gets 10^n where n = number of digits in mantissa
func (z *Cash) minorUnitFactor() int64 {
	return MinorUnit[z.FracDigits]
}

// sets the precision to the right of the decimal point (mantissa)
// call before String() to get custom precision with proper rounding
func (z *Cash) SetPrec(prec int) {
	z.FracDigits = prec
}

// can we do math between these two `Cash` instances?
func (z *Cash) isCompatible(x *Cash) bool {
	if z.FracDigits != x.FracDigits || z.Currency != x.Currency || z.Decimal != x.Decimal || z.Thousands != x.Thousands {
		return false
	}
	return true
}

// rounds an integer half-to-even—like IEEE 754 does
// strips "last," least significant digit (e.g., 3 in 123)
// least significant digit determines direction of rounding
// please: try to avoid rounding! this is money!
func roundLikeBankers(x int64) int64 {
	var (
		leastSigDigit int64 = x % 10
		mostSigDigits int64 = x / 10
	)

	switch {
	case leastSigDigit < 5:
		return mostSigDigits
	case leastSigDigit > 5:
		return mostSigDigits + 1
	case leastSigDigit == 5:
		return mostSigDigits + (mostSigDigits & 1)
	default:
		// won't happen but compiler is stupid
		return 0
	}
}

// SetString() on already allocated `Cash`
func (z *Cash) SetString(src string) (*Cash, error) {
	var (
		parts = strings.Split(src, string(z.Decimal))
		err   error
	)

	switch len(parts) {
	case 1: // just an integer
		z.Amt, err = strconv.ParseInt(src, 10, 64)
		if err != nil {
			return nil, err
		}
		return z, nil
	case 2: // decimal
		integerPart, err := strconv.ParseInt(parts[0], 10, 64)
		if err != nil {
			return nil, err
		}
		integerPart *= z.minorUnitFactor()

		// sanitize fractional part
		fracPartLen := utf8.RuneCountInString(parts[1])
		if fracPartLen > z.FracDigits {
			// just leave one extra digit for rounding
			parts[1] = parts[1][:z.FracDigits+1]
		}
		fracPart, err := strconv.ParseInt(parts[1], 10, 64)
		if err != nil {
			return nil, err
		}
		if fracPartLen > z.FracDigits {
			// handle rounding for mantissa
			fracPart = roundLikeBankers(fracPart)
		}
		z.Amt = integerPart + fracPart
		return z, nil
	default:
		return nil, ErrBadString
	}
}

// set the value of the minor unit
// calling it cents just so you know what I mean
func (z *Cash) SetCents(cents int64) *Cash {
	z.Amt = cents
	return z
}

// String()
func (z *Cash) String() string {
	var (
		buf         bytes.Buffer
		integerPart string
		fracPart    string
		neg         bool
	)

	if z.IsPositive() != true {
		neg=true
		z.Amt = z.Amt * -1 // make positive
		buf.WriteString("(")
	}

	buf.WriteRune(z.Currency) // dollar sign
	// decimal
	decRaw := strconv.FormatInt(z.Amt, 10)
	decRawLen := utf8.RuneCountInString(decRaw)

	// is the int string too small? (that's what she said)
	// outcome graph for the integer and fractional parts
	switch {
	case decRawLen == 0:
		// totally empty
		return ""

	case decRawLen == 1:
		// [0, 9] cents—it's one digit
		integerPart = "0"
		fracPart = "0" + decRaw

	case decRawLen == 2:
		// only fractional, sans integer part
		// 0.##
		integerPart = "0"
		fracPart = decRaw

	case decRawLen > z.FracDigits:
		// init integer part
		integerPart = decRaw[:decRawLen-z.FracDigits]
		// apply digit grouping on each thousands
		integerPart = commafy(integerPart, z.Thousands)
		// init fractional part
		fracPart = decRaw[decRawLen-z.FracDigits:]
	}

	// now build the overall string
	buf.WriteString(integerPart) // write left side of decimal pt
	buf.WriteRune(z.Decimal)     // decimal point
	buf.WriteString(fracPart)    // write right side of decimal pt

	if neg {
		buf.WriteString(")")
		z.Amt = z.Amt * -1 // make negative
	}

	return buf.String()
}

// commafy string of digits; digit grouping by thousands
func commafy(s string, comma rune) string {
	var (
		l   = utf8.RuneCountInString(s)
		q   = l / 3
		m   = l % 3
		pos int
		buf bytes.Buffer
	)
	buf.WriteString(s[0:m])
	for i := 0; i < q; i++ {
		buf.WriteRune(comma)
		pos = m + i
		buf.WriteString(s[pos : pos+3])
	}
	return buf.String()
}

// TODO NewFromFloat64

// NewFromBigRat
func (z *Cash) NewFromBigRat(src *big.Rat) (*Cash, error) {
	s := src.FloatString(z.FracDigits)
	return z.SetString(s)
}

// get big.Rat representation
func (z *Cash) Rat() *big.Rat {
	return big.NewRat(z.Amt, z.minorUnitFactor())
}

// addition
func (z *Cash) Add(x, y *Cash) (*Cash, error) {
	if !z.isCompatible(x) || !z.isCompatible(y) {
		return nil, ErrIncompatible
	}
	z.Amt = x.Amt + y.Amt
	return z, nil
}

// subtraction
func (z *Cash) Sub(x, y *Cash) (*Cash, error) {
	if !z.isCompatible(x) || !z.isCompatible(y) {
		return nil, ErrIncompatible
	}
	z.Amt = x.Amt - y.Amt
	return z, nil
}

// multiply `Cash` with a scalar value
// e.g., $18.18 * 5
// most realistic use case of multiplication for `Cash`
func (z *Cash) MulByScalar(x *Cash, scalar int64) (*Cash, error) {
	if !z.isCompatible(x) {
		return nil, ErrIncompatible
	}
	z.Amt = x.Amt * scalar
	return z, nil
}

// multiply `Cash` with a rational number
// under the hood: math/big.Rat
// has mathematical accuracy
// good for consecutive mul (or div) operations
func (z *Cash) MulByRat(x *Cash, p *big.Rat) (*Cash, error) {
	if !z.isCompatible(x) {
		return nil, ErrIncompatible
	}

	// turn integer cents to a rational number
	var xR *big.Rat
	if x.Rational == nil {
		xR = big.NewRat(x.Amt, x.minorUnitFactor())
	} else {
		xR = x.Rational
	}

	// multiply fractions
	z.Rational = new(big.Rat).Mul(xR, p)

	// retrieve integer cents
	// TODO this is slow as shit—restructure code to avoid this
	s := z.Rational.FloatString(z.FracDigits)
	_, err := z.SetString(s)
	if err != nil {
		return nil, err
	}

	return z, nil
}

// multiplying two `Cash` money values
// seems unlikely to be used at all
// this is only here because it would look stupid if it weren't here
func (z *Cash) MulByCash(x, y *Cash) (*Cash, error) {
	if !z.isCompatible(x) || !z.isCompatible(y) {
		return nil, ErrIncompatible
	}
	z.Amt = (x.Amt * y.Amt) / z.minorUnitFactor()
	return z, nil
}

// divide `Cash` by a scalar integer N
// return a slice of N respective `Cash` values
// inspired by Martin Fowler's "allocate"
func (z *Cash) DivByScalar(y int64) []Cash {
	var (
		i      int64
		minima int64  = z.Amt / y
		maxima int64  = minima + 1
		mod    int64  = z.Amt % y
		ret    []Cash = make([]Cash, y) // guarantee: y > mod
	)

	// first, assign maxima to res
	// because sum(maxima - minima) over [0, mod) is less than 1
	for i = 0; i < mod; i++ {
		ret[i] = *z.SetCents(maxima) // keeping results consistent/compatible with input
	}

	// then, assign minima to leftovers in res
	for i = mod; i < y; i++ {
		ret[i] = *z.SetCents(minima)
	}

	return ret
}

// divide `Cash` according to a set of numbers representing a ratio
// return a slice of `Cash` values as long as the set (ratio)
// inspired by Martin Fowler's "allocate"
func (z *Cash) DivIntoRatio(ratio []int64) []Cash {
	var (
		l           int    = len(ratio)
		ret         []Cash = make([]Cash, l)
		denominator int64
		t           int64
	)

	for i := 0; i < l; i++ {
		denominator += ratio[i] // summing parts of `ratio`
	}

	mod := z.Amt // start with whole; before subtracting
	for j := 0; j < l; j++ {
		t = z.Amt * ratio[j] / denominator
		ret[j] = *z // shallow copy the context `Cash`
		ret[j].SetCents(t)
		mod -= t // ...eventually, actual modulus
	}

	// use up the modulus by adding 1, starting from i=0
	var i int64 = 0
	for ; i < mod; i++ {
		ret[i].Amt += 1
	}

	return ret
}

// database serialization
func (z *Cash) Value() (driver.Value, error) {
	return z.String(), nil
}

// database deserialization
func (z *Cash) Scan(src interface{}) error {
	switch src := src.(type) {
	case int64:
		// treat as cents
		t := NewUSD().SetCents(src) // TODO come on, USD as default, really...?
		*z = *t
		return nil

		// TODO float64

	default:
		// treat as string
		// works for MySQL
		// check if quoted; if so, remove quotes
		b := src.(string)
		if len(b) > 2 && b[0] == '"' && b[len(b)-1] == '"' {
			b = b[1 : len(b)-1]
		}
		t, err := NewUSD().SetString(b) // TODO generalize, not USD by default
		*z = *t
		return err
	}

	return nil
}

// json.Marshaler interface impl
func (z *Cash) MarshalJSON() ([]byte, error) {
	s := "\"" + z.String() + "\"" // add quotes
	return []byte(s), nil
}

// json.Unmarshaler interface impl
func (z *Cash) UnmarshalJSON(b []byte) error {
	// check if `b` is quoted; if so, unquote
	if len(b) > 2 && b[0] == '"' && b[len(b)-1] == '"' {
		b = b[1 : len(b)-1]
	}
	// output from `b`
	t, err := NewUSD().SetString(string(b))
	if err != nil {
		return err
	}
	*z = *t    // copy memory
	return nil // fin
}

// classic comparison
func (z *Cash) Cmp(y *Cash) (int, error) {
	if !z.isCompatible(y) {
		return -2, ErrIncompatible
	}

	switch {
	case z.Amt < y.Amt:
		return -1, nil
	case z.Amt == y.Amt:
		return 0, nil
	case z.Amt > y.Amt:
		return 1, nil
	}

	return -2, nil
}

// is greater than
func (z *Cash) IsGreaterThan(y *Cash) (bool, error) {
	r, err := z.Cmp(y)
	return r == 1, err
}

// equals
func (z *Cash) Equals(y *Cash) (bool, error) {
	r, err := z.Cmp(y)
	return r == 0, err
}

// is less than
func (z *Cash) IsLessThan(y *Cash) (bool, error) {
	r, err := z.Cmp(y)
	return r == -1, err
}

func (z *Cash) IsPositive() bool {
	return z.Amt > 0
}

// errors
var (
	ErrBadString    = errors.New("malformed input string")
	ErrIncompatible = errors.New("Cash values have incompatible fields")
	ErrCannotScan   = errors.New("Scan() failed: Cannot convert passed value to data type")
)
