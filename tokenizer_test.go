package ion

import (
	"io"
	"testing"
)

func TestNext(t *testing.T) {
	tok := tokenizeString("foo::'foo':[] 123, {})")

	next := func(tt tokenType) {
		if err := tok.Next(); err != nil {
			t.Fatal(err)
		}
		if tok.Token() != tt {
			t.Fatalf("expected %v, got %v", tt, tok.Token())
		}
	}

	next(tokenSymbol)
	next(tokenDoubleColon)
	next(tokenSymbolQuoted)
	next(tokenColon)
	next(tokenOpenBracket)
	next(tokenNumeric)
	next(tokenComma)
	next(tokenOpenBrace)
}

func TestIsTripleQuote(t *testing.T) {
	test := func(str string, eok bool, next int) {
		t.Run(str, func(t *testing.T) {
			tok := tokenizeString(str)

			ok, err := tok.isTripleQuote()
			if err != nil {
				t.Fatal(err)
			}
			if ok != eok {
				t.Errorf("expected ok=%v, got ok=%v", eok, ok)
			}

			read(t, tok, next)
		})
	}

	test("''string'''", true, 's')
	test("'string'''", false, '\'')
	test("'", false, '\'')
	test("", false, -1)
}

func TestIsInf(t *testing.T) {
	test := func(str string, eok bool, next int) {
		t.Run(str, func(t *testing.T) {
			tok := tokenizeString(str)
			c, err := tok.read()
			if err != nil {
				t.Fatal(err)
			}

			ok, err := tok.isInf(c)
			if err != nil {
				t.Fatal(err)
			}

			if ok != eok {
				t.Errorf("expected %v, got %v", eok, ok)
			}

			c, err = tok.read()
			if err != nil {
				t.Fatal(err)
			}
			if c != next {
				t.Errorf("expected '%c', got '%c'", next, c)
			}
		})
	}

	test("+inf", true, -1)
	test("-inf", true, -1)
	test("+inf ", true, ' ')
	test("-inf\t", true, '\t')
	test("-inf\n", true, '\n')
	test("+inf,", true, ',')
	test("-inf}", true, '}')
	test("+inf)", true, ')')
	test("-inf]", true, ']')
	test("+inf//", true, '/')
	test("+inf/*", true, '/')

	test("+inf/", false, 'i')
	test("-inf/0", false, 'i')
	test("+int", false, 'i')
	test("-iot", false, 'i')
	test("+unf", false, 'u')
	test("_inf", false, 'i')

	test("-in", false, 'i')
	test("+i", false, 'i')
	test("+", false, -1)
	test("-", false, -1)
}

func TestScanForNumericType(t *testing.T) {
	test := func(str string, ett tokenType) {
		t.Run(str, func(t *testing.T) {
			tok := tokenizeString(str)
			c, err := tok.read()
			if err != nil {
				t.Fatal(err)
			}

			tt, err := tok.scanForNumericType(c)
			if err != nil {
				t.Fatal(err)
			}
			if tt != ett {
				t.Errorf("expected %v, got %v", ett, tt)
			}
		})
	}

	test("0b0101", tokenBinary)
	test("0B", tokenBinary)
	test("0xABCD", tokenHex)
	test("0X", tokenHex)
	test("0000-00-00", tokenTimestamp)
	test("0000T", tokenTimestamp)

	test("0", tokenNumeric)
	test("1b0101", tokenNumeric)
	test("1B", tokenNumeric)
	test("1x0101", tokenNumeric)
	test("1X", tokenNumeric)
	test("1234", tokenNumeric)
	test("12345", tokenNumeric)
	test("1,23T", tokenNumeric)
	test("12,3T", tokenNumeric)
	test("123,T", tokenNumeric)
}

func TestSkipWhitespace(t *testing.T) {
	test := func(str string, eok bool, ec int) {
		t.Run(str, func(t *testing.T) {
			tok := tokenizeString(str)
			c, ok, err := tok.skipWhitespace()
			if err != nil {
				t.Fatal(err)
			}

			if ok != eok {
				t.Errorf("expected ok=%v, got ok=%v", eok, ok)
			}
			if c != ec {
				t.Errorf("expected c='%c', got c='%c'", ec, c)
			}
		})
	}

	test("/ 0)", false, '/')
	test("xyz_", false, 'x')
	test(" / 0)", true, '/')
	test(" xyz_", true, 'x')
	test(" \t\r\n / 0)", true, '/')
	test("\t\t  // comment\t\r\n\t\t  x", true, 'x')
	test(" \r\n /* comment *//* \r\n comment */x", true, 'x')
}

func TestSkipLobWhitespace(t *testing.T) {
	test := func(str string, eok bool, ec int) {
		t.Run(str, func(t *testing.T) {
			tok := tokenizeString(str)
			c, ok, err := tok.skipLobWhitespace()
			if err != nil {
				t.Fatal(err)
			}

			if ok != eok {
				t.Errorf("expected ok=%v, got ok=%v", eok, ok)
			}
			if c != ec {
				t.Errorf("expected c='%c', got c='%c'", ec, c)
			}
		})
	}

	test("///=", false, '/')
	test("xyz_", false, 'x')
	test(" ///=", true, '/')
	test(" xyz_", true, 'x')
	test("\r\n\t///=", true, '/')
	test("\r\n\txyz_", true, 'x')
}

func TestSkipCommentsHandler(t *testing.T) {
	t.Run("SingleLine", func(t *testing.T) {
		tok := tokenizeString("/comment\nok")
		ok, err := tok.skipCommentsHandler()
		if err != nil {
			t.Fatal(err)
		}
		if !ok {
			t.Error("expected ok=true, got ok=false")
		}

		read(t, tok, 'o')
		read(t, tok, 'k')
		read(t, tok, -1)
	})

	t.Run("Block", func(t *testing.T) {
		tok := tokenizeString("*comm\nent*/ok")
		ok, err := tok.skipCommentsHandler()
		if err != nil {
			t.Fatal(err)
		}
		if !ok {
			t.Error("expected ok=true, got ok=false")
		}

		read(t, tok, 'o')
		read(t, tok, 'k')
		read(t, tok, -1)
	})

	t.Run("FalseAlarm", func(t *testing.T) {
		tok := tokenizeString(" 0)")
		ok, err := tok.skipCommentsHandler()
		if err != nil {
			t.Fatal(err)
		}
		if ok {
			t.Error("expected ok=false, got ok=true")
		}

		read(t, tok, ' ')
		read(t, tok, '0')
		read(t, tok, ')')
		read(t, tok, -1)
	})
}

func TestSkipSingleLineComment(t *testing.T) {
	tok := tokenizeString("single-line comment\r\nok")
	err := tok.skipSingleLineComment()
	if err != nil {
		t.Fatal(err)
	}

	read(t, tok, 'o')
	read(t, tok, 'k')
	read(t, tok, -1)
}

func TestSkipSingleLineCommentOnLastLine(t *testing.T) {
	tok := tokenizeString("single-line comment")
	err := tok.skipSingleLineComment()
	if err != nil {
		t.Fatal(err)
	}

	read(t, tok, -1)
}

func TestSkipBlockComment(t *testing.T) {
	tok := tokenizeString("this is/ a\nmulti-line /** comment.**/ok")
	err := tok.skipBlockComment()
	if err != nil {
		t.Fatal(err)
	}

	read(t, tok, 'o')
	read(t, tok, 'k')
	read(t, tok, -1)
}

func TestSkipInvalidBlockComment(t *testing.T) {
	tok := tokenizeString("this is a comment that never ends")
	err := tok.skipBlockComment()
	if err == nil {
		t.Error("did not fail on bad block comment")
	}
}

func TestPeekN(t *testing.T) {
	tok := tokenizeString("abc\r\ndef")

	peekN(t, tok, 1, nil, 'a')
	peekN(t, tok, 2, nil, 'a', 'b')
	peekN(t, tok, 3, nil, 'a', 'b', 'c')

	read(t, tok, 'a')
	read(t, tok, 'b')

	peekN(t, tok, 3, nil, 'c', '\n', 'd')
	peekN(t, tok, 2, nil, 'c', '\n')
	peekN(t, tok, 3, nil, 'c', '\n', 'd')

	read(t, tok, 'c')
	read(t, tok, '\n')
	read(t, tok, 'd')

	peekN(t, tok, 3, io.EOF, 'e', 'f')
	peekN(t, tok, 3, io.EOF, 'e', 'f')
	peekN(t, tok, 2, nil, 'e', 'f')

	read(t, tok, 'e')
	read(t, tok, 'f')
	read(t, tok, -1)

	peekN(t, tok, 10, io.EOF)
}

func peekN(t *testing.T, tok *tokenizer, n int, ee error, ecs ...int) {
	cs, err := tok.peekN(n)
	if err != ee {
		t.Fatalf("expected err=%v, got err=%v", ee, err)
	}
	if !equal(ecs, cs) {
		t.Errorf("expected %v, got %v", ecs, cs)
	}
}

func equal(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}

	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}

	return true
}

func TestPeek(t *testing.T) {
	tok := tokenizeString("abc")

	peek(t, tok, 'a')
	peek(t, tok, 'a')
	read(t, tok, 'a')

	peek(t, tok, 'b')
	tok.unread('a')

	peek(t, tok, 'a')
	read(t, tok, 'a')
	read(t, tok, 'b')
	peek(t, tok, 'c')
	peek(t, tok, 'c')

	read(t, tok, 'c')
	peek(t, tok, -1)
	peek(t, tok, -1)
	read(t, tok, -1)
}

func peek(t *testing.T, tok *tokenizer, expected int) {
	c, err := tok.peek()
	if err != nil {
		t.Fatal(err)
	}
	if c != expected {
		t.Errorf("expected %v, got %v", expected, c)
	}
}

func TestReadUnread(t *testing.T) {
	tok := tokenizeString("abc\rd\ne\r\n")

	read(t, tok, 'a')
	tok.unread('a')

	read(t, tok, 'a')
	read(t, tok, 'b')
	read(t, tok, 'c')
	tok.unread('c')
	tok.unread('b')

	read(t, tok, 'b')
	read(t, tok, 'c')
	read(t, tok, '\n')
	tok.unread('\n')

	read(t, tok, '\n')
	read(t, tok, 'd')
	read(t, tok, '\n')
	read(t, tok, 'e')
	read(t, tok, '\n')
	read(t, tok, -1)

	tok.unread(-1)
	tok.unread('\n')

	read(t, tok, '\n')
	read(t, tok, -1)
	read(t, tok, -1)
}

func read(t *testing.T, tok *tokenizer, expected int) {
	c, err := tok.read()
	if err != nil {
		t.Fatal(err)
	}
	if c != expected {
		t.Errorf("expected %v, got %v", expected, c)
	}
}