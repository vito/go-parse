package main

import (
	"fmt";
	"unicode";

	. "./parsec";
)

type aCall struct {
	target	string;
	args	[]Output;
}

func call(in Vessel) (c Output, ok bool) {
	name, ok := Identifier(in);
	if !ok {
		fmt.Printf("FAILED. %#v\n", name);
		return;
	}
	fmt.Printf("PASSED. %#v\n", name);

	call, ok := Parens(Many(Identifier))(in);
	c = aCall{name.(string), call.([]interface{})};
	return;
}

func main() {
	in := new(StringVessel);
	in.SetInput(`< (>)((
<
))( (  (>)))  < >
>

    >`);

	in.SetSpec(Spec{
		"/*",
		"*/",
		"//",
		true,
		Satisfy(unicode.IsLower),
		Satisfy(unicode.IsLetter),
		Satisfy(func(c int) bool { return !unicode.IsLetter(c) }),
		Satisfy(func(c int) bool { return !unicode.IsLetter(c) }),
		[]string{"func"},
		[]string{"+", "-"},
		true,
	});

	ltgt := Any(Symbol("<"), Symbol(">"));

	var parser Parser;
	parser = Many(Any(ltgt, Parens(R(&parser))));
	out, parsed := parser(in);

	fmt.Printf("Matched: %#v\n", parsed);
	fmt.Printf("Matches: %v\n", out);
	fmt.Printf("Vessel: %+v\n", in);

	try := Try(All(Symbol("<"), Parens(Symbol(">")), Symbol("("), Symbol("FAIL")));
	in.SetPosition(Position{});
	out, parsed = try(in);

	fmt.Printf("\nTry results:\n");
	fmt.Printf("Matched: %#v\n", parsed);
	fmt.Printf("Matches: %+v\n", out);
	fmt.Printf("Vessel: %+v\n", in);

	in.SetInput("foo(bar)");
	in.SetPosition(Position{});

	out, parsed = call(in);
	fmt.Printf("\nCall results:\n");
	fmt.Printf("Matched: %#v\n", parsed);
	fmt.Printf("Matches: %+v\n", out);
	fmt.Printf("Vessel: %+v\n", in);
}
