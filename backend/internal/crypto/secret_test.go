package crypto

import "testing"

func TestSealOpen(t *testing.T) {
	b, err := NewBox("test-master-key-32bytes-min!!")
	if err != nil {
		t.Fatal(err)
	}
	ct, err := b.Seal(`{"access":"tok"}`)
	if err != nil {
		t.Fatal(err)
	}
	if ct == `{"access":"tok"}` {
		t.Fatal("not encrypted")
	}
	pt, err := b.Open(ct)
	if err != nil {
		t.Fatal(err)
	}
	if pt != `{"access":"tok"}` {
		t.Fatalf("pt=%s", pt)
	}
}
