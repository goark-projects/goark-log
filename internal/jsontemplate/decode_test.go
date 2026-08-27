package jsontemplate

import "testing"

func TestDecodeRawFields_whenObjectProvided_shouldPreserveOrder(t *testing.T) {
	fields, err := DecodeRawFields(`{"a":{"$resolver":"message"},"b":true}`)
	if err != nil {
		t.Fatalf("DecodeRawFields() error = %v", err)
	}
	if len(fields) != 2 || fields[0].Key != "a" || fields[1].Key != "b" {
		t.Fatalf("fields = %+v, want ordered a,b", fields)
	}
	if string(fields[1].Raw) != "true" {
		t.Fatalf("field b raw = %s, want true", fields[1].Raw)
	}
}

func TestDecodeRawFields_whenTrailingTokenProvided_shouldReject(t *testing.T) {
	_, err := DecodeRawFields(`{"a":true} []`)
	if err == nil {
		t.Fatalf("DecodeRawFields() should reject trailing token")
	}
}
