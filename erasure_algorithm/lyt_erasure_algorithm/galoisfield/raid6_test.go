package galoisfield

import (
	"fmt"
	"math/rand"
	"testing"
)

func TestRaid6(t *testing.T) {
	enc, _ := Raid6New(3, 2)
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
	err := enc.Encode(data)
	if err != nil {
		fmt.Println("Error:", err)
	}
	fmt.Println("data:", data)

	data[1] = nil
	data[3] = nil
	fmt.Println("data:", data)
	err = enc.ReconstructData(data)
	fmt.Println("data:", data)

	bigfile := make([]byte, 10000)
	split, err := enc.Split(bigfile)
	if err != nil {
		panic(err)
	}
	print(split[1])
}
