package parsec

import (
	"container/vector";
	/*"fmt";*/
	"reflect";
	"unicode";
)


// Container of the input, position, and any user/parser state.
type Vessel interface {
	GetState() State;
	SetState(State);

	GetInput() Input;
	SetInput(Input);

	GetPosition() Position;
	SetPosition(Position);

	GetSpec() Spec;
	SetSpec(Spec);

	Get(int) (Input, bool);
	Next() (int, bool);
	Pop(int);
	Push(int);
}

// Specifications for the parser
type Spec struct {
	CommentStart	string;
	CommentEnd	string;
	CommentLine	string;
	NestedComments	bool;
	IdentStart	Parser;
	IdentLetter	Parser;
	OpStart		Parser;
	OpLetter	Parser;
	ReservedNames	[]Output;
	ReservedOpNames	[]Output;
	CaseSensitive	bool;
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
	Name	string;
	Line	int;
	Column	int;
	Offset	int;
}


// Token that satisfies a condition.
func Satisfy(check func(c int) bool) Parser {
	return func(in Vessel) (Output, bool) {
		target, ok := in.Next();
		if ok && check(target) {
			in.Pop(1);
			return target, true;
		}

		return nil, false;
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
			Many(Satisfy(func(c int) bool { return c != '\n' }))))(in);
	}
}

func MultiLineComment() Parser {
	return func(in Vessel) (Output, bool) {
		spec := in.GetSpec();

		return Skip(All(
			String(spec.CommentStart),
			InComment()))(in);
	}
}

func InComment() Parser {
	return func(in Vessel) (Output, bool) {
		if in.GetSpec().NestedComments {
			return inMulti()(in)
		}

		return inSingle()(in);
	}
}

func inMulti() Parser {
	return func(in Vessel) (Output, bool) {
		spec := in.GetSpec();
		startEnd := spec.CommentStart + spec.CommentEnd;

		return Any(
			Try(String(spec.CommentEnd)),
			All(MultiLineComment(), inMulti()),
			All(Many1(NoneOf(startEnd)), inMulti()),
			All(OneOf(startEnd), inMulti()))(in);
	}
}

func inSingle() Parser {
	return func(in Vessel) (Output, bool) {
		spec := in.GetSpec();
		startEnd := spec.CommentStart + spec.CommentEnd;

		return Any(
			Try(String(spec.CommentEnd)),
			All(Many1(NoneOf(startEnd)), inSingle()),
			All(OneOf(startEnd), inSingle()))(in);
	}
}

func OneOf(cs string) Parser {
	return func(in Vessel) (Output, bool) {
		next, ok := in.Next();
		if !ok {
			return nil, false
		}

		for _, v := range cs {
			if v == next {
				in.Pop(1);
				return v, true;
			}
		}

		return next, false;
	}
}

func NoneOf(cs string) Parser {
	return func(in Vessel) (Output, bool) {
		next, ok := in.Next();
		if !ok {
			return nil, false
		}

		for _, v := range cs {
			if v == next {
				return v, false
			}
		}

		in.Pop(1);
		return next, true;
	}
}

func Skip(match Parser) Parser {
	return func(in Vessel) (Output, bool) {
		_, ok := match(in);
		return nil, ok;
	}
}

func Token() Parser {
	return func(in Vessel) (next Output, ok bool) {
		next, ok = in.Next();
		in.Pop(1);
		return;
	}
}

// Match a parser and skip whitespace
func Lexeme(match Parser) Parser {
	return func(in Vessel) (Output, bool) {
		out, matched := match(in);
		if !matched {
			return nil, false
		}

		Whitespace()(in);

		return out, true;
	}
}

// Match a parser 0 or more times.
func Many(match Parser) Parser {
	return func(in Vessel) (Output, bool) {
		matches := new(vector.Vector);
		for {
			out, parsed := match(in);
			if !parsed {
				break
			}

			if out != nil {
				matches.Push(out)
			}
		}

		return matches.Data(), true;
	}
}

func Many1(match Parser) Parser {
	return func(in Vessel) (Output, bool) {
		a, ok := match(in);
		if !ok {
			return nil, false
		}

		rest, ok := Many(match)(in);
		if !ok {
			return nil, false
		}

		as := rest.([]interface{});

		all := make([]interface{}, len(as)+1);
		all[0] = a;
		for i := 0; i < len(as); i++ {
			all[i+1] = as[i]
		}

		return all, true;
	}
}

// Match a parser seperated by another parser 0 or more times.
// Trailing delimeters are valid.
func SepBy(delim Parser, match Parser) Parser {
	return func(in Vessel) (Output, bool) {
		matches := new(vector.Vector);
		for {
			out, parsed := match(in);
			if !parsed {
				break
			}

			matches.Push(out);

			_, sep := delim(in);
			if !sep {
				break
			}
		}

		return matches, true;
	}
}

// Go through the parsers until one matches.
func Any(parsers ...) Parser {
	return func(in Vessel) (Output, bool) {
		p := reflect.NewValue(parsers).(*reflect.StructValue);

		for i := 0; i < p.NumField(); i++ {
			parser := p.Field(i).Interface().(Parser);
			match, ok := parser(in);
			if ok {
				return match, ok
			}
		}

		return nil, false;
	}
}

// Match all parsers, returning the final result. If one fails, it stops.
// NOTE: Consumes input on failure. Wrap calls in Try(...) to avoid.
func All(parsers ...) Parser {
	return func(in Vessel) (match Output, ok bool) {
		p := reflect.NewValue(parsers).(*reflect.StructValue);

		for i := 0; i < p.NumField(); i++ {
			parser := p.Field(i).Interface().(Parser);
			match, ok = parser(in);
			if !ok {
				return
			}
		}

		return;
	}
}

// Match all parsers, collecting their outputs into a vector.
// If one parser fails, the whole thing fails.
// NOTE: Consumes input on failure. Wrap calls in Try(...) to avoid.
func Collect(parsers ...) Parser {
	return func(in Vessel) (Output, bool) {
		p := reflect.NewValue(parsers).(*reflect.StructValue);

		matches := new(vector.Vector);
		for i := 0; i < p.NumField(); i++ {
			parser := p.Field(i).Interface().(Parser);
			match, ok := parser(in);
			if !ok {
				return nil, false
			}

			matches.Push(match);
		}

		return matches, true;
	}
}

// Try matching begin, match, and then end.
func Between(begin Parser, end Parser, match Parser) Parser {
	return func(in Vessel) (Output, bool) {
		parse, ok := Try(Collect(begin, match, end))(in);
		if !ok {
			return nil, false
		}

		return parse.(*vector.Vector).At(1), true;
	}
}

// Lexeme parser for `match' wrapped in parens.
func Parens(match Parser) Parser	{ return Lexeme(Between(Symbol("("), Symbol(")"), match)) }

// Match a string and consume any following whitespace.
func Symbol(str string) Parser	{ return Lexeme(String(str)) }

// Match a string and pop the string's length from the input.
// NOTE: Consumes input on failure. Wrap calls in Try(...) to avoid.
func String(str string) Parser {
	return func(in Vessel) (Output, bool) {
		for _, v := range str {
			next, ok := in.Next();
			if !ok || next != v {
				return nil, false
			}

			in.Pop(1);
		}

		return str, true;
	}
}

// Try a parse and revert the state and position if it fails.
func Try(match Parser) Parser {
	return func(in Vessel) (Output, bool) {
		st, pos := in.GetState(), in.GetPosition();
		out, ok := match(in);
		if !ok {
			in.SetState(st);
			in.SetPosition(pos);
		}

		return out, ok;
	}
}

func Ident() Parser {
	return func(in Vessel) (name Output, ok bool) {
		sp := in.GetSpec();
		n, ok := sp.IdentStart(in);
		if !ok {
			return
		}

		ns, ok := Many(sp.IdentLetter)(in);
		if !ok {
			return
		}

		rest := make([]int, len(ns.([]interface{})));
		for k, v := range ns.([]interface{}) {
			rest[k] = v.(int)
		}

		return string(n.(int)) + string(rest), true;
	}
}

func Identifier() Parser {
	return Lexeme(Try(func(in Vessel) (name Output, ok bool) {
		name, ok = Ident()(in);
		if !ok {
			return
		}

		for _, v := range in.GetSpec().ReservedNames {
			if v == name {
				return nil, false
			}
		}

		return;
	}))
}


// Basic string vessel for parsing over a string input.
type StringVessel struct {
	state		State;
	input		string;
	position	Position;
	spec		Spec;
}

func (self *StringVessel) GetState() State	{ return self.state }

func (self *StringVessel) SetState(st State)	{ self.state = st }

func (self *StringVessel) GetInput() Input {
	i := 0;
	for o, _ := range self.input {
		if i == self.position.Offset {
			return self.input[o:]
		}
		i++;
	}

	return "";
}

func (self *StringVessel) Get(i int) (Input, bool) {
	if len(self.input) < self.position.Offset+i {
		return "", false
	}

	s := "";
	n := 0;
	for _, v := range self.input {
		if n >= self.position.Offset {
			if n > self.position.Offset+i {
				break
			}
			s += string(v);
		}
		n++;
	}

	return s, true;
}

func (self *StringVessel) Next() (int, bool) {
	if len(self.input) < self.position.Offset+1 {
		return 0, false
	}

	i := 0;
	for _, v := range self.input {
		if i == self.position.Offset {
			return int(v), true
		}
		i++;
	}

	return 0, false;
}

func (self *StringVessel) Pop(i int)	{ self.position.Offset += i }

func (self *StringVessel) Push(i int)	{ self.position.Offset -= i }

func (self *StringVessel) SetInput(in Input)	{ self.input = in.(string) }

func (self *StringVessel) GetPosition() Position {
	return self.position
}

func (self *StringVessel) SetPosition(pos Position) {
	self.position = pos
}

func (self *StringVessel) GetSpec() Spec	{ return self.spec }

func (self *StringVessel) SetSpec(sp Spec)	{ self.spec = sp }
