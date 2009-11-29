package main

import (
    "container/vector";
	"fmt";
    "strings";
    "unicode";

	. "./parsec";
)

type rChar struct {
    codepoint int;
    str string;
}
type rGroup struct {
    target interface{};
}
type rOption struct {
    target interface{};
}
type rStar struct {
    target interface{};
}


func isSpecial(char int) bool {
    switch char {
        case '(', ')', '[', ']', '?', '^', '*', '.':
            return true;
    }

    return false;
}

func isNotSpecial(char int) bool {
    return !isSpecial(char);
}

func grouped(match Parser) Parser {
    return func(in Vessel) (c Output, ok bool) {
        call, ok := Between(String("("), String(")"), match)(in);
        if !ok {
            return;
        }

        c = rGroup{call};
        return;
    }
}

var normal Parser = func(in Vessel) (Output, bool) {
    next, ok := in.Next();
    if !ok || isSpecial(next) {
        return nil, false
    }

    if next == '\\' {
        in.Pop(1);
        next, ok = in.Next();
    }

    if !ok {
        return nil, false
    }

    in.Pop(1);

    return rChar{next, string(next)}, true
}

var optional Parser = func(in Vessel) (Output, bool) {
    result, ok := Collect(Any(normal, grouped(regexp)), String("?"))(in);
    if !ok {
        return nil, false
    }

    return rOption{result.(*vector.Vector).At(0)}, true;
}

var star Parser = func(in Vessel) (Output, bool) {
    result, ok := Collect(Any(normal, grouped(regexp)), String("*"))(in);

    if !ok {
        return nil, false
    }

    return rStar{result.(*vector.Vector).At(0)}, true;
}

var regexp Parser = func(in Vessel) (Output, bool) {
    return Many(Any(Identifier(), Try(star), Try(optional), Skip(All(OneLineComment(), String("\n"))), MultiLineComment(), Try(normal), grouped(R(&regexp))))(in);
}

// A hacked-together monstrosity that pretty-prints any complex
// structure with indenting and whitespace and such.
func pretty(thing interface{}) (s string) {
    in := fmt.Sprintf("%#v\n", thing);

    indent := 0;
    inString := false;
    for i, char := range in {
        if !inString || char == '"' {
            switch char {
            case ',':
                s += string(char) + "\n" + strings.Repeat("    ", indent);
            case '(', '{':
                if in[i+2] != '}' {
                    indent++;
                    s += string(char) + "\n" + strings.Repeat("    ", indent);
                } else {
                    s += "{}";
                }
            case ')', '}':
                if in[i-2] != '{' {
                    indent--;
                    s += "\n" + strings.Repeat("    ", indent) + string(char);
                }
            case ':':
                s += ": ";
            case ' ':
                if in[i-1] != ',' && in[i-9 : i] != "interface" {
                    s += " ";
                }
            case '"':
                inString = !inString;
                fallthrough;
            default:
                s += string(char);
            }
        } else {
            s += string(char);
        }
    }

    return
}

func main() {
	in := new(StringVessel);
    spec := Spec{};
    spec.CommentLine = "--";
    spec.CommentStart = "{-";
    spec.CommentEnd = "-}";
    spec.NestedComments = true;
    spec.IdentStart = Satisfy(unicode.IsUpper);
    spec.IdentLetter = Satisfy(unicode.IsLower);
    spec.ReservedNames = []Output{"Foo"};
    in.SetSpec(spec);

    in.SetInput(`a 日本語 \[\]\({- test -} ( b)?ccc*-- comment
l*{- foo {- {- test -} -}-}Bar Foo FizzBuzz`);

    fmt.Printf("Parsing `%s`...\n", in.GetInput());

	out, parsed := regexp(in);

	fmt.Printf("Parsed: %#v\n", parsed);
	fmt.Printf("Tree: %s\n", pretty(out));
    fmt.Printf("Rest: %#v\n", in.GetInput());
}
