package main

import (
	"fmt"
	"log"
	"os"

	binparsergen "github.com/benoitkugler/textlayout/fonts/bin-parser-gen"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatal("missing input file")
	}
	input := os.Args[1]
	err := binparsergen.Generate(input)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Done.")
}
