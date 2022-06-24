package main

import (
	"fmt"
	"math/rand"

	"github.com/klauspost/reedsolomon"
)

func main() {
	enc, err := reedsolomon.New(3, 2)
	data := make([][]byte, 5)

	for i := range data {
		data[i] = make([]byte, 3)
	}

	for _, in := range data[:3] {
		for j := range in {
			in[j] = byte(rand.Intn(255))
		}
	}

	fmt.Println("data:", data)
	err = enc.Encode(data)
	if err != nil {
		fmt.Println("Error:", err)
	}
	fmt.Println("data:", data)

	data[0] = nil
	data[1] = nil
	fmt.Println("data:", data)
	err = enc.Reconstruct(data)
	fmt.Println("data:", data)

	// bigfile := make([]byte, 10000)
	// split, err := enc.Split(bigfile)
	// if err != nil {
	// 	panic(err)
	// }
	// print(split[1])
}
