package main

import (
	"fmt"

	"example.com/i"
)

func main() {
	i.Hello()
	a := i.New(4, 0x7, 2)
	b := i.New(4, 0x7, 2)
	if a == b {
		fmt.Println("expected singleton, got multiple instances of %#v", a)
	}

	// a := byte(1)
	// b := byte(2)
	// c := i.Add(a, b)
	// fmt.Println("The result is c", c)
}
