package proxy

import "testing"

func TestParseLineHTTPAuth(t *testing.T) {
	e, err := ParseLine("http://user:secret@1.2.3.4:8080")
	if err != nil {
		t.Fatal(err)
	}
	if e.Scheme != "http" || e.Host != "1.2.3.4" || e.Port != 8080 || e.Username != "user" || e.Password != "secret" {
		t.Fatalf("%+v", e)
	}
	if contains(e.Display(), "secret") {
		t.Fatalf("display leaked password: %s", e.Display())
	}
}

func TestParseLineDefaultScheme(t *testing.T) {
	e, err := ParseLine("user:pass@10.0.0.1:3128")
	if err != nil {
		t.Fatal(err)
	}
	if e.Scheme != "http" || e.Host != "10.0.0.1" || e.Port != 3128 {
		t.Fatalf("%+v", e)
	}
}

func TestParseLineSocks5(t *testing.T) {
	e, err := ParseLine("socks5://127.0.0.1:1080")
	if err != nil {
		t.Fatal(err)
	}
	if e.Scheme != "socks5" || e.Port != 1080 {
		t.Fatalf("%+v", e)
	}
}

func TestParseLineComment(t *testing.T) {
	e, err := ParseLine("# comment")
	if err != nil || e != nil {
		t.Fatalf("e=%v err=%v", e, err)
	}
}

func TestParseLines(t *testing.T) {
	text := `
# pool
http://a:b@1.1.1.1:80
socks5://2.2.2.2:1080
badline
`
	entries, errs := ParseLines(text)
	if len(entries) != 2 {
		t.Fatalf("entries=%d errs=%v", len(entries), errs)
	}
	if len(errs) != 1 {
		t.Fatalf("errs=%v", errs)
	}
}

func contains(s, sub string) bool {
	return sub != "" && (len(s) >= len(sub)) && (stringIndex(s, sub) >= 0)
}

func stringIndex(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
