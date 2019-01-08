package parsec

import (
	"unicode"
)

// Container of the input, position, and any user/parser state.
type Vessel interface {
	GetState() State
	SetState(State)

	GetInput() Input
	SetInput(Input)

	GetPosition() Position
	SetPosition(Position)

	GetSpec() Spec
	SetSpec(Spec)

	Get(int) (Input, bool)
	Next() (rune, bool)
	Pop(int)
	Push(int)
}

// Specifications for the parser
type Spec struct {
	CommentStart    string
	CommentEnd      string
	CommentLine     string
	NestedComments  bool
	IdentStart      Parser
	IdentLetter     Parser
	OpStart         Parser
	OpLetter        Parser
	ReservedNames   []Output
	ReservedOpNames []Output
	CaseSensitive   bool
}

// A Parser is a function that takes a vessel and returns any matches
// (Output) and whether or not the match was valid.
type Parser func(Vessel) (Output, bool)

// Input type used by vessels
type Input interface{}

// Output of Parsers
type Output interface{}

// Any value can be a vessel's state.
type State interface{}

// Position in the input.
type Position struct {
	Name   string
	Line   int
	Column int
	Offset int
}

// Token that satisfies a condition.
func Satisfy(check func(c rune) bool) Parser {
	return func(in Vessel) (Output, bool) {
		target, ok := in.Next()
		if ok && check(target) {
			in.Pop(1)
			return target, true
		}

		return nil, false
	}
}

// Skip whitespace and comments
func Whitespace() Parser {
	return Many(Any(Satisfy(unicode.IsSpace), OneLineComment(), MultiLineComment()))
}

func OneLineComment() Parser {
	return func(in Vessel) (Output, bool) {
		if in.GetSpec().CommentLine == "" {
			return nil, false
		}

		return Skip(All(
			Try(String(in.GetSpec().CommentLine)),
			Many(Satisfy(func(c rune) bool { return c != '\n' }))))(in)
	}
}

func MultiLineComment() Parser {
	return func(in Vessel) (Output, bool) {
		spec := in.GetSpec()

		return Skip(All(
			String(spec.CommentStart),
			InComment()))(in)
	}
}

func InComment() Parser {
	return func(in Vessel) (Output, bool) {
		if in.GetSpec().NestedComments {
			return inMulti()(in)
		}

		return inSingle()(in)
	}
}

func inMulti() Parser {
	return func(in Vessel) (Output, bool) {
		spec := in.GetSpec()
		startEnd := spec.CommentStart + spec.CommentEnd

		return Any(
			Try(String(spec.CommentEnd)),
			All(MultiLineComment(), inMulti()),
			All(Many1(NoneOf(startEnd)), inMulti()),
			All(OneOf(startEnd), inMulti()))(in)
	}
}

func inSingle() Parser {
	return func(in Vessel) (Output, bool) {
		spec := in.GetSpec()
		startEnd := spec.CommentStart + spec.CommentEnd

		return Any(
			Try(String(spec.CommentEnd)),
			All(Many1(NoneOf(startEnd)), inSingle()),
			All(OneOf(startEnd), inSingle()))(in)
	}
}

func OneOf(cs string) Parser {
	return func(in Vessel) (Output, bool) {
		next, ok := in.Next()
		if !ok {
			return nil, false
		}

		for _, v := range cs {
			if v == next {
				in.Pop(1)
				return v, true
			}
		}

		return next, false
	}
}

func NoneOf(cs string) Parser {
	return func(in Vessel) (Output, bool) {
		next, ok := in.Next()
		if !ok {
			return nil, false
		}

		for _, v := range cs {
			if v == next {
				return v, false
			}
		}

		in.Pop(1)
		return next, true
	}
}

func Skip(match Parser) Parser {
	return func(in Vessel) (Output, bool) {
		_, ok := match(in)
		return nil, ok
	}
}

func Token() Parser {
	return func(in Vessel) (next Output, ok bool) {
		next, ok = in.Next()
		in.Pop(1)
		return
	}
}

// Match a parser and skip whitespace
func Lexeme(match Parser) Parser {
	return func(in Vessel) (Output, bool) {
		out, matched := match(in)
		if !matched {
			return nil, false
		}

		Whitespace()(in)

		return out, true
	}
}

// Match a parser 0 or more times.
func Many(match Parser) Parser {
	return func(in Vessel) (Output, bool) {
		var matches []interface{}
		for {
			out, parsed := match(in)
			if !parsed {
				break
			}

			if out != nil {
				matches = append(matches, out)
			}
		}

		return matches, true
	}
}

func Many1(match Parser) Parser {
	return func(in Vessel) (Output, bool) {
		a, ok := match(in)
		if !ok {
			return nil, false
		}

		rest, ok := Many(match)(in)
		if !ok {
			return nil, false
		}

		as := rest.([]interface{})

		all := make([]interface{}, len(as)+1)
		all[0] = a
		for i := 0; i < len(as); i++ {
			all[i+1] = as[i]
		}

		return all, true
	}
}

// Match a parser seperated by another parser 0 or more times.
// Trailing delimeters are valid.
func SepBy(delim Parser, match Parser) Parser {
	return func(in Vessel) (Output, bool) {
		var matches []interface{}
		for {
			out, parsed := match(in)
			if !parsed {
				break
			}

			matches = append(matches, out)

			_, sep := delim(in)
			if !sep {
				break
			}
		}

		return matches, true
	}
}

// Go through the parsers until one matches.
func Any(parsers ...Parser) Parser {
	return func(in Vessel) (Output, bool) {
		for _, parser := range parsers {
			match, ok := parser(in)
			if ok {
				return match, ok
			}
		}

		return nil, false
	}
}

// Match all parsers, returning the final result. If one fails, it stops.
// NOTE: Consumes input on failure. Wrap calls in Try(...) to avoid.
func All(parsers ...Parser) Parser {
	return func(in Vessel) (match Output, ok bool) {
		for _, parser := range parsers {
			match, ok = parser(in)
			if !ok {
				return
			}
		}

		return
	}
}

// Match all parsers, collecting their outputs into a vector.
// If one parser fails, the whole thing fails.
// NOTE: Consumes input on failure. Wrap calls in Try(...) to avoid.
func Collect(parsers ...Parser) Parser {
	return func(in Vessel) (Output, bool) {
		var matches []interface{}
		for _, parser := range parsers {
			match, ok := parser(in)
			if !ok {
				return nil, false
			}

			matches = append(matches, match)
		}

		return matches, true
	}
}

// Try matching begin, match, and then end.
func Between(begin Parser, end Parser, match Parser) Parser {
	return func(in Vessel) (Output, bool) {
		parse, ok := Try(Collect(begin, match, end))(in)
		if !ok {
			return nil, false
		}

		return parse.([]interface{})[1], true
	}
}

// Lexeme parser for `match' wrapped in parens.
func Parens(match Parser) Parser { return Lexeme(Between(Symbol("("), Symbol(")"), match)) }

// Match a string and consume any following whitespace.
func Symbol(str string) Parser { return Lexeme(String(str)) }

// Match a string and pop the string's length from the input.
// NOTE: Consumes input on failure. Wrap calls in Try(...) to avoid.
func String(str string) Parser {
	return func(in Vessel) (Output, bool) {
		for _, v := range str {
			next, ok := in.Next()
			if !ok || next != v {
				return nil, false
			}

			in.Pop(1)
		}

		return str, true
	}
}

// Try a parse and revert the state and position if it fails.
func Try(match Parser) Parser {
	return func(in Vessel) (Output, bool) {
		st, pos := in.GetState(), in.GetPosition()
		out, ok := match(in)
		if !ok {
			in.SetState(st)
			in.SetPosition(pos)
		}

		return out, ok
	}
}

func Ident() Parser {
	return func(in Vessel) (name Output, ok bool) {
		sp := in.GetSpec()
		n, ok := sp.IdentStart(in)
		if !ok {
			return
		}

		ns, ok := Many(sp.IdentLetter)(in)
		if !ok {
			return
		}

		rest := make([]rune, len(ns.([]interface{})))
		for k, v := range ns.([]interface{}) {
			rest[k] = v.(rune)
		}

		return string(n.(rune)) + string(rest), true
	}
}

func Identifier() Parser {
	return Lexeme(Try(func(in Vessel) (name Output, ok bool) {
		name, ok = Ident()(in)
		if !ok {
			return
		}

		for _, v := range in.GetSpec().ReservedNames {
			if v == name {
				return nil, false
			}
		}

		return
	}))
}

// Basic string vessel for parsing over a string input.
type StringVessel struct {
	state    State
	input    string
	position Position
	spec     Spec
}

func (s *StringVessel) GetState() State { return s.state }

func (s *StringVessel) SetState(st State) { s.state = st }

func (s *StringVessel) GetInput() Input {
	var i int
	for o := range s.input {
		if i == s.position.Offset {
			return s.input[o:]
		}
		i++
	}

	return ""
}

func (s *StringVessel) Get(i int) (Input, bool) {
	if len(s.input) < s.position.Offset+i {
		return "", false
	}

	var (
		n   int
		str string
	)

	for _, v := range s.input {
		if n >= s.position.Offset {
			if n > s.position.Offset+i {
				break
			}
			str += string(v)
		}
		n++
	}

	return s, true
}

func (s *StringVessel) Next() (rune, bool) {
	if len(s.input) < s.position.Offset+1 {
		return 0, false
	}

	var i int

	for _, v := range s.input {
		if i == s.position.Offset {
			return rune(v), true
		}
		i++
	}

	return 0, false
}

func (s *StringVessel) Pop(i int) { s.position.Offset += i }

func (s *StringVessel) Push(i int) { s.position.Offset -= i }

func (s *StringVessel) SetInput(in Input) { s.input = in.(string) }

func (s *StringVessel) GetPosition() Position {
	return s.position
}

func (s *StringVessel) SetPosition(pos Position) {
	s.position = pos
}

func (s *StringVessel) GetSpec() Spec { return s.spec }

func (s *StringVessel) SetSpec(sp Spec) { s.spec = sp }
