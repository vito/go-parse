package main

import (
	"fmt"
	"strings"
	"unicode"

	. "github.com/vito/go-parse"
)

type rChar struct {
	char rune
	str  string
}
type rToken struct {
	char rune
	str  string
}
type rGroup struct {
	target interface{}
}
type rOption struct {
	target interface{}
}
type rStar struct {
	target interface{}
}

func isMeta(char rune) bool {
	switch char {
	case '(', ')', '[', ']', '?', '^', '*', '.', '+', '$', '|':
		return true
	}

	return false
}

func isSpecial(char rune) bool {
	switch char {
	case 'a', 'b', 'f', 'n', 'r', 't', 'v', '\\':
		return true
	case 'w', 's', 'd':
		return true
	}

	return false
}

func isNotMeta(char rune) bool { return !isMeta(char) }

func isNotSpecial(char rune) bool { return !isSpecial(char) }

func special(char rune) Output {
	if isNotSpecial(char) {
		return nil
	}

	switch char {
	case '\\':
		return rChar{'\\', "\\"}
	case 'a', 'b', 'f':
		return rChar{char - rune(90), string(char - rune(90))}
	case 'n':
		return rChar{'\n', "\n"}
	case 'r':
		return rChar{'\r', "\r"}
	case 't':
		return rChar{'\t', "\t"}
	case 'v':
		return rChar{'\v', "\v"}
	case 'w', 's', 'd':
		return rToken{char, string(char)}
	}

	return nil
}

func grouped(match Parser) Parser {
	return func(in Vessel) (c Output, ok bool) {
		call, ok := Between(String("("), String(")"), match)(in)
		if !ok {
			return
		}

		c = rGroup{call}
		return
	}
}

func char() Parser {
	return func(in Vessel) (out Output, ok bool) {
		next, ok := Satisfy(isNotMeta)(in)
		if !ok {
			return
		}

		char := next.(rune)
		if char == '\\' {
			next, ok = Satisfy(func(c rune) bool { return isMeta(c) || isSpecial(c) })(in)
			if ok {
				char = next.(rune)
				if isSpecial(char) {
					out = special(char)
				}
			}
		} else {
			out = rChar{char, string(char)}
		}

		return
	}
}

func optional() Parser {
	return func(in Vessel) (Output, bool) {
		result, ok := Collect(Any(char(), grouped(regexp())), String("?"))(in)
		if !ok {
			return nil, false
		}

		return rOption{result.([]interface{})[0]}, true
	}
}

func star() Parser {
	return func(in Vessel) (Output, bool) {
		result, ok := Collect(Any(char(), grouped(regexp())), String("*"))(in)

		if !ok {
			return nil, false
		}

		return rStar{result.([]interface{})[0]}, true
	}
}

func regexp() Parser {
	return func(in Vessel) (Output, bool) {
		return Many(
			Any(
				Identifier(),
				Try(star()),
				Try(optional()),
				Skip(All(OneLineComment(), String("\n"))),
				MultiLineComment(),
				Try(char()),
				grouped(regexp())))(in)
	}
}

// A hacked-together monstrosity that pretty-prints any complex
// structure with indenting and whitespace and such.
func pretty(thing interface{}) (s string) {
	in := fmt.Sprintf("%#v\n", thing)

	indent := 0
	inString := false
	for i, char := range in {
		if !inString || char == '"' {
			switch char {
			case ',':
				s += string(char) + "\n" + strings.Repeat("    ", indent)
			case '(', '{':
				if in[i+2] != '}' {
					indent++
					s += string(char) + "\n" + strings.Repeat("    ", indent)
				} else {
					s += "{}"
				}
			case ')', '}':
				if in[i-2] != '{' {
					indent--
					s += "\n" + strings.Repeat("    ", indent) + string(char)
				}
			case ':':
				s += ": "
			case ' ':
				if in[i-1] != ',' && in[i-9:i] != "interface" {
					s += " "
				}
			case '"':
				inString = !inString
				fallthrough
			default:
				s += string(char)
			}
		} else {
			s += string(char)
		}
	}

	return
}

func main() {
	in := new(StringVessel)

	in.SetSpec(Spec{
		CommentStart:   "{-",
		CommentEnd:     "-}",
		CommentLine:    "--",
		NestedComments: true,
		IdentStart:     Satisfy(unicode.IsUpper),
		IdentLetter:    Satisfy(unicode.IsLower),
		ReservedNames:  []Output{"Foo"},
		CaseSensitive:  true,
	})

	in.SetInput(`a 日本語 \[\]\({- test -} ( b)?ccc*-- comment
l*{- foo {- {- test -} -}-}Bar FooFizz
Buzz\a\n\t\f\w\s\d*`)

	fmt.Printf("Parsing `%s`...\n", in.GetInput())

	out, ok := regexp()(in)

	if _, unfinished := in.Next(); unfinished {
		fmt.Printf("Incomplete parse: %s\n", pretty(out))
		fmt.Println("Parse error.")
		fmt.Printf("Position: %+v\n", in.GetPosition())
		fmt.Printf("State: %+v\n", in.GetState())
		fmt.Printf("Rest: `%s`\n", in.GetInput())
		return
	}

	fmt.Printf("Parsed: %#v\n", ok)
	fmt.Printf("Tree: %s\n", pretty(out))
	fmt.Printf("Rest: %#v\n", in.GetInput())
}
