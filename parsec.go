package parsec

import (
    "container/vector";
    "reflect";
    "strings";
)


// Container of the input, position, and any user/parser state.
type Vessel interface {
    GetState() State;
    SetState(State);

    GetInput() Input;
    SetInput(Input);

    GetPosition() Position;
    SetPosition(Position);

    Get(int) Input;
    Pop(int);
    Push(int);
}

// A Parser is a function that takes a vessel and returns any matches
// (Output) and whether or not the match was valid.
type Parser func (Vessel) (Output, bool)

// Input type used by vessels
type Input interface{}

// Output of Parsers
type Output interface{}

// Any value can be a vessel's state.
type State interface{}

// Position in the input.
type Position struct {
    Name    string;
    Line    int;
    Column  int;
    Offset  int;
}


func IsSpace(c byte) bool {
    switch c {
    case ' ', '\t', '\n', '\r', '\f', '\v', '\xa0':
        return true;
    }

    return false;
}

// Token that satisfies a condition.
func Satisfy(check func(c byte) bool) Parser {
    return func(in Vessel) (Output, bool) {
        target := in.Get(1);
        if target != nil && check(target.(string)[0]) {
            in.Pop(1);
            return target, true;
        }

        return nil, false
    }
}

// Skip whitespace (TODO: Comments)
func Whitespace(in Vessel) (Output, bool) {
    return Many(Satisfy(IsSpace))(in);
}

// Match a parser and skip whitespace
func Lexeme(match Parser) Parser {
    return func(in Vessel) (Output, bool) {
        out, matched := match(in);
        Whitespace(in);
        return out, matched
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

            matches.Push(out);
        }

        return matches.Data(), true
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

        return nil, false
    }
}

func Between(begin Parser, end Parser, match Parser) Parser {
    return func(in Vessel) (Output, bool) {
        before := in.GetPosition();

        _, ok := begin(in);
        if !ok {
            return nil, false
        }

        out, ok := match(in);
        if !ok {
            in.SetPosition(before);
            return nil, false
        }

        _, ok = end(in);
        if !ok {
            return nil, false
        }

        return out, true
    }
}

func Parens(match Parser) Parser {
    return Lexeme(Between(Symbol("("), Symbol(")"), match));
}

func Symbol(str string) Parser {
    return func(in Vessel) (Output, bool) {
        match, ok := String(str)(in);
        if !ok {
            return nil, false
        }

        Whitespace(in);

        return match, ok;
    }
}

// Match a string and pop the string's length from the input.
func String(str string) Parser {
    return func(in Vessel) (Output, bool) {
        if strings.HasPrefix(in.GetInput().(string), str) {
            in.Pop(len(str));
            return str, true;
        }

        return nil, false;
    }
}


// Basic string vessel for parsing over a string input.
type StringVessel struct {
    state State;
    input string;
    position Position;
}

func (self *StringVessel) GetState() State {
    return self.state;
}

func (self *StringVessel) SetState(st State) {
    self.state = st;
}

func (self *StringVessel) GetInput() Input {
    return self.input[self.position.Offset:];
}

func (self *StringVessel) Get(i int) Input {
    if len(self.input) < self.position.Offset + i {
        return nil;
    }

    return self.input[self.position.Offset:self.position.Offset + i];
}

func (self *StringVessel) Pop(i int) {
    self.position.Offset += i;
}

func (self *StringVessel) Push(i int) {
    self.position.Offset -= i;
}

func (self *StringVessel) SetInput(in Input) {
    self.input = in.(string);
}

func (self *StringVessel) GetPosition() Position {
    return self.position;
}

func (self *StringVessel) SetPosition(pos Position) {
    self.position = pos;
}

