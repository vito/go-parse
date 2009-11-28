package main

import (
        "fmt";

        . "./parsec";
)

func main() {
        in := new(StringVessel);
        in.SetInput(`< (>)((
<
))( (  (>)))  < >
>

    >`);

        ltgt := Any(Symbol("<"), Symbol(">"));

        var parser Parser;
        parser = Many(Any(ltgt, Parens(R(&parser))));
        out, parsed := parser(in);

        fmt.Printf("Matched: %#v\n", parsed);
        fmt.Printf("Matches: %v\n", out);
        fmt.Printf("Vessel: %+v\n", in);

        try := Try(All(Symbol("<"), Parens(Symbol(">")), Symbol("("), Symbol("(")));
        in.SetPosition(Position{});
        out, parsed = try(in);

        fmt.Printf("\nTry results:\n");
        fmt.Printf("Matched: %#v\n", parsed);
        fmt.Printf("Matches: %#v\n", out);
        fmt.Printf("Vessel: %+v\n", in);
}
