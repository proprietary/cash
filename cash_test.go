package cash

import (
	"github.com/stretchr/testify/assert"
	"log"
	"math/big"
	"testing"
)

func TestCreateFromString(t *testing.T) {
	a, err := New(USD).SetString("12.392")
	if err != nil {
		log.Fatal(err)
	}
	assert.EqualValues(t, 1239, a.Amt, "should be equal")
}

func TestAdd(t *testing.T) {
	d := New(USD).SetCents(1023)
	e := New(USD).SetCents(9920)
	f, err := New(USD).Add(d, e)
	if err != nil {
		log.Fatal(err)
	}
	assert.EqualValues(t, 10943, f.Amt, "should equal")
}

func TestSub(t *testing.T) {
	a, err := New(USD).SetString("18.2123")
	if err != nil {
		log.Fatal(err)
	}
	b, err := New(USD).SetString("1.02")
	if err != nil {
		log.Fatal(err)
	}
	c, err := New(USD).Sub(a, b)
	if err != nil {
		log.Fatal(err)
	}
	assert.EqualValues(t, 1719, c.Amt, "should equal")
}

func TestMakeString(t *testing.T) {
	g := New(USD).SetCents(1001897)
	gString := g.String()
	assert.EqualValues(t, "$10,018.97", gString, "should equal")
}

func TestMakeStringTenths(t *testing.T) {
	g := NewUSD().SetCents(8)
	gString := g.String()
	assert.EqualValues(t, "$0.08", gString, "should equal")
}

func TestRounding(t *testing.T) {
	a, err := NewUSD().SetString("666.995")
	if err != nil {
		log.Fatal(err)
	}
	assert.EqualValues(t, 66700, a.Amt, "should equal")
}

func TestMulByScalar(t *testing.T) {
	a := NewUSD().SetCents(9022)
	var scalar int64 = 6
	b, err := NewUSD().MulByScalar(a, scalar)
	assert.Nil(t, err)
	assert.EqualValues(t, 54132, b.Amt, "90.22 * 6 == 541.32; should equal!")
}

func TestMulByFraction(t *testing.T) {
	a := NewUSD().SetCents(1818)
	rat := big.NewRat(3, 4)
	b, err := NewUSD().MulByRat(a, rat)
	assert.Nil(t, err)
	assert.EqualValues(t, 1364, b.Amt, "18.18 * 3/4 == 13.64")
}

func TestMulByCash(t *testing.T) {
	a := NewUSD().SetCents(1818)
	b := NewUSD().SetCents(1717)
	c, err := NewUSD().MulByCash(a, b)
	assert.Nil(t, err)
	assert.EqualValues(t, 31215, c.Amt, "18.18 * 17.17 == 312.15")
}

func TestDivByScalar(t *testing.T) {
	a := NewUSD().SetCents(100)
	var scalar int64 = 3
	res := a.DivByScalar(scalar)

	assert.EqualValues(t, 34, res[0].Amt)
	assert.EqualValues(t, 33, res[1].Amt)
	assert.EqualValues(t, 33, res[2].Amt)

	assert.True(t, len(res) == int(scalar), "length of res should be same as 'scalar' denominator")
}

func TestDivIntoRatio(t *testing.T) {
	a := NewUSD().SetCents(100)
	ratio := []int64{1, 1, 1}
	res := a.DivIntoRatio(ratio)

	/*
		for i, v := range res {
			log.Printf("%d\t%v\n", i, v.String())
		}
	*/

	assert.True(t, len(res) == len(ratio))

	assert.EqualValues(t, 34, res[0].Amt)
	assert.EqualValues(t, 33, res[1].Amt)
	assert.EqualValues(t, 33, res[2].Amt)
}

func TestScan(t *testing.T) {
	q := new(Cash)
	var s string = "55.10"
	err := q.Scan(s)
	if err != nil {
		t.Error("Failed on string:", err)
	}
	assert.EqualValues(t, 5510, q.Amt, "should equal")

	w := new(Cash)
	var ii int64 = 6629
	err = w.Scan(ii)
	if err != nil {
		t.Error("Failed on cents; int64:", err)
	}
	assert.EqualValues(t, ii, w.Amt, "should equal")
}

func TestCmp(t *testing.T) {
	a, err := NewUSD().SetString("25.60")
	assert.Nil(t, err)
	b, err := NewUSD().SetString("18.40")
	assert.Nil(t, err)
	c, err := a.Cmp(b)
	assert.Nil(t, err)
	assert.EqualValues(t, 1, c, "should equal")

	is, err := a.IsGreaterThan(b)
	assert.Nil(t, err)
	assert.EqualValues(t, true, is, "a > b so a.IsGreaterThan(b) == true")

	is, err = a.IsLessThan(b)
	assert.Nil(t, err)
	assert.EqualValues(t, false, is, "a > b so a.IsLessThan(b) == false")

	is, err = a.Equals(b)
	assert.Nil(t, err)
	assert.EqualValues(t, false, is, "a != b so a.Equals(b) == false")
}

func TestRoundTrip(t *testing.T) {
	expected := New(USD).SetCents(1001897)
	actual, err := New(USD).SetString(expected.String())
	assert.Nil(t, err)
	assert.EqualValues(t, expected.Amt, actual.Amt)

	expected = New(USD).SetCents(-1001897)
	actual, err = New(USD).SetString(expected.String())
	assert.Nil(t, err)
	assert.EqualValues(t, expected.Amt, actual.Amt)
}