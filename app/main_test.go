package main

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFormat(t *testing.T) {
	tbl := []struct {
		inp  string
		out  string
		size int
	}{
		{"blah", "blah", 100},
		{"blah <p>xyx</p>", "blah xyx", 100},
		{"blah <p>xyx</p> something 122 abcdefg 12345 qwer", "blah xyx ...", 15},
		{"blah <p>xyx</p> something 122 abcdefg 12345 qwerty", "blah xyx something ...", 20},
		{"<p>xyx</p><title>", "xyx", 20},
	}

	for i, tt := range tbl {
		t.Run(fmt.Sprintf("check-%d", i), func(t *testing.T) {
			out := format(tt.inp, tt.size)
			assert.Equal(t, tt.out, out)
		})
	}
}
