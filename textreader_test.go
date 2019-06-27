package ion

import (
	"errors"
	"fmt"
	"math"
	"math/big"
	"testing"
)

func TestInts(t *testing.T) {
	test := func(str string, m func(Reader) error) {
		t.Run(str, func(t *testing.T) {
			r := NewTextReaderString(str)
			if !r.Next() {
				t.Error("next returned false")
			}
			if r.Type() != IntType {
				t.Errorf("expected type=IntType, got %v", r.Type())
			}

			if err := m(r); err != nil {
				t.Error(err)
			}

			if r.Next() {
				t.Error("next returned true")
			}
			if r.Err() != nil {
				t.Error(r.Err())
			}
		})
	}

	test("null.int", func(r Reader) error {
		if !r.IsNull() {
			return errors.New("expected isnull=true, got false")
		}

		val, err := r.IntValue()
		if err != nil {
			return err
		}
		if val != 0 {
			return fmt.Errorf("expected 0, got %v", val)
		}

		return nil
	})

	testInt := func(str string, eval int) {
		test(str, func(r Reader) error {
			val, err := r.IntValue()
			if err != nil {
				return err
			}
			if val != eval {
				return fmt.Errorf("expected %v, got %v", eval, val)
			}
			return nil
		})
	}

	testInt("0", 0)
	testInt("12345", 12345)
	testInt("-12345", -12345)
	testInt("0b000101", 5)
	testInt("-0b000101", -5)
	testInt("0x01020e0F", 0x01020e0f)
	testInt("-0x01020e0F", -0x01020e0f)

	testInt64 := func(str string, eval int64) {
		test(str, func(r Reader) error {
			val, err := r.Int64Value()
			if err != nil {
				return err
			}
			if val != eval {
				return fmt.Errorf("expected %v, got %v", eval, val)
			}
			return nil
		})
	}

	testInt64("0x123FFFFFFFF", 0x123FFFFFFFF)
	testInt64("-0x123FFFFFFFF", -0x123FFFFFFFF)

	testBigInt := func(str string) {
		test(str, func(r Reader) error {
			val, err := r.BigIntValue()
			if err != nil {
				return err
			}

			eval, _ := (&big.Int{}).SetString(str, 0)
			if eval.Cmp(val) != 0 {
				return fmt.Errorf("expected %v, got %v", eval, val)
			}

			return nil
		})
	}

	testBigInt("0xEFFFFFFFFFFFFFFF")
	testBigInt("0xFFFFFFFFFFFFFFFF")
	testBigInt("-0x1FFFFFFFFFFFFFFFF")
}

func TestStrings(t *testing.T) {
	r := NewTextReaderString(`foo::"bar" "baz" 'a'::'b'::'''beep''' '''boop''' null.string`)

	test := func(etas []string, eval string) {
		if !r.Next() {
			t.Fatal("next returned false")
		}

		if r.Type() != StringType {
			t.Fatalf("expected type=string, got type=%v", r.Type())
		}

		if !strequals(r.TypeAnnotations(), etas) {
			t.Errorf("expected tas=%v, got tas=%v", etas, r.TypeAnnotations())
		}

		val, err := r.StringValue()
		if err != nil {
			t.Fatal(err)
		}

		if val != eval {
			t.Errorf("expected val=%v, got val=%v", eval, val)
		}
	}

	test([]string{"foo"}, "bar")
	test(nil, "baz")
	test([]string{"a", "b"}, "beepboop")
	test(nil, "")

	if r.Next() {
		t.Errorf("next unexpectedly returned true")
	}
	if r.Err() != nil {
		t.Error(r.Err())
	}
}

func TestSymbols(t *testing.T) {
	r := NewTextReaderString("'null'::foo bar a::b::'baz' null.symbol")

	test := func(etas []string, eval string) {
		if !r.Next() {
			t.Fatal("next returned false")
		}

		if r.Type() != SymbolType {
			t.Fatalf("expected type=symbol, got type=%v", r.Type())
		}

		if !strequals(r.TypeAnnotations(), etas) {
			t.Errorf("expected tas=%v, got tas=%v", etas, r.TypeAnnotations())
		}

		val, err := r.StringValue()
		if err != nil {
			t.Fatal(err)
		}

		if val != eval {
			t.Errorf("expected val=%v, got val=%v", eval, val)
		}
	}

	test([]string{"null"}, "foo")
	test(nil, "bar")
	test([]string{"a", "b"}, "baz")
	test(nil, "")

	if r.Next() {
		t.Errorf("next unexpectedly returned true")
	}
	if r.Err() != nil {
		t.Error(r.Err())
	}
}

func strequals(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}

	for i := 0; i < len(a); i++ {
		if a[i] != b[i] {
			return false
		}
	}

	return true
}

func TestSpecialSymbols(t *testing.T) {
	r := NewTextReaderString("null\nnull.struct\ntrue\nfalse\nnan")

	// null
	{
		if !r.Next() {
			t.Fatal("next returned false")
		}
		if r.Type() != NullType {
			t.Errorf("expected type=NullType, got %v", r.Type())
		}
		if !r.IsNull() {
			t.Error("expected isNull=true, got false")
		}
	}

	// null.struct
	{
		if !r.Next() {
			t.Fatal("next returned false")
		}
		if r.Type() != StructType {
			t.Errorf("expected type=StructType, got %v", r.Type())
		}
		if !r.IsNull() {
			t.Error("expected isNull=true, got false")
		}
	}

	// true
	{
		if !r.Next() {
			t.Fatal("next returned false")
		}
		if r.Type() != BoolType {
			t.Errorf("expected type=BoolType, got %v", r.Type())
		}
		val, err := r.BoolValue()
		if err != nil {
			t.Fatal(err)
		}
		if !val {
			t.Error("expected value=true, got false")
		}
	}

	// false
	{
		if !r.Next() {
			t.Fatal("next returned false")
		}
		if r.Type() != BoolType {
			t.Errorf("expected type=BoolType, got %v", r.Type())
		}
		val, err := r.BoolValue()
		if err != nil {
			t.Fatal(err)
		}
		if val {
			t.Error("expected value=false, got true")
		}
	}

	// nan
	{
		if !r.Next() {
			t.Fatal("next returned false")
		}
		if r.Type() != FloatType {
			t.Errorf("expected type=FloatType, got %v", r.Type())
		}
		val, err := r.FloatValue()
		if err != nil {
			t.Fatal(err)
		}
		if !math.IsNaN(val) {
			t.Errorf("expected value=NaN, got %v", val)
		}
	}

	if r.Next() {
		t.Error("next returned true")
	}
}