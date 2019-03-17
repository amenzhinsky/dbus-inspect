package main

import (
	"testing"
)

func TestFmtArgType(t *testing.T) {
	noColorFlag = true
	for sig, want := range map[string]string{
		"?":           "Unknown(?)",
		"n?n":         "Int16, Unknown(?), Int16",
		"nn":          "Int16, Int16",
		"nqiuxt":      "Int16, Uint16, Int32, Uint32, Int64, Uint64",
		"i(iy)":       "Int32, Struct(Int32, Byte)",
		"a{o(i(uu))}": "Dict{Object, Struct(Int32, Struct(Uint32, Uint32))}",
		"a{(uu)s}":    "Malformed(a{(uu)s})",
		"ai":          "Array[Int32]",
		"aai":         "Array[Array[Int32]]",
		"aa{yy}":      "Array[Dict{Byte, Byte}]",
	} {
		if have := fmtArgType(sig); have != want {
			t.Errorf("fmtArgType(%q) = %q, want %q", sig, have, want)
		}
	}
}
