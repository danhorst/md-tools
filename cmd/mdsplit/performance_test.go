package main

import (
	"strconv"
	"strings"
	"testing"
)

func BenchmarkUnmatchedBrackets(b *testing.B) {
	for _, n := range []int{40000, 80000, 160000} {
		input := strings.Repeat("[", n)
		b.Run(strconv.Itoa(n), func(b *testing.B) {
			b.SetBytes(int64(n))
			for i := 0; i < b.N; i++ {
				splitSentences(input)
			}
		})
	}
}
