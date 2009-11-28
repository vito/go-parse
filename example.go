package main

import (
    "fmt";

    . "./parsec";
)

func main() {
    in := new(StringVessel);
    in.SetInput(`< (>)(
<
)(  >)  < >
>

    >`);

    ltgt := Any(Symbol("<"), Symbol(">"));

    parser := Many(Any(ltgt, Parens(ltgt)));
    out, parsed := parser(in);

    fmt.Printf("Matched: %#v\n", parsed);
    fmt.Printf("Matches: %v\n", out);
    fmt.Printf("Vessel: %+v\n", in);
}
