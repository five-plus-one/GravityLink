package base62

import "testing"

func TestEncode(t *testing.T) {
	cases := map[uint64]string{
		0:    "0",
		1:    "1",
		9:    "9",
		10:   "a",
		35:   "z",
		36:   "A",
		61:   "Z",
		62:   "10",
		3843: "ZZ",
	}

	for input, expected := range cases {
		if actual := Encode(input); actual != expected {
			t.Fatalf("Encode(%d) = %q, want %q", input, actual, expected)
		}
	}
}
