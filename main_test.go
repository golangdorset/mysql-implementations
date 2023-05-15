package main

import (
	"testing"

	_ "github.com/go-sql-driver/mysql"
)

func TestMetadata_Value(t *testing.T) {
	Metadata := Metadata{
		KV{
			
			Val: "world",
		},
	}

	val, err := Metadata.Value()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	t.Log(string(val.([]byte)))
}
