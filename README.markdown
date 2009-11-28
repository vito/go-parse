Go Parse

A [Parsec](http://hackage.haskell.org/package/parsec-3.0.0)-like library for Go.

Structure:

A Vessel is what carries around the input and any user-specified state, as well as internal state such as the position in the input. It should know how to return and set those 3 values, as well as Get from the input, and push/pop (which just adjusts the position).

A Parser takes a Vessel and returns an Output and whether or not the parse was successful.

Parsers can typically be combined in many ways. For example Symbol is just String followed by Whitespace, Many takes a Parser and repeatedly applies it, matching 0 or more times (thus, the parse is always successful), and Any takes any number of Parsers and tries them all in order until one succeeds.

Example:

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

Output:

    go-parse $ go parsec
    Matched: true
    Matches: [< > < > < > > >]
    Vessel: &{state:<nil> input:< (>)(
    <
    )(  >)  < >
    >

        > position:{Name: Line:0 Column:0 Offset:29}}

