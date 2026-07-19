package cloak

import (
	"context"
	"testing"

	"github.com/zninggo/grokforge/internal/domain"
)

func TestDryRunRegister(t *testing.T) {
	e := &Engine{DryRun: true}
	if !e.Available() {
		t.Fatal("dry run should be available")
	}
	sess, err := e.Open(context.Background(), domain.BrowserOpenOptions{
		ProxyURI: "http://user:secret@1.2.3.4:8080",
		Headless: true,
		Humanize: true,
		GeoIP:    true,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer sess.Close(context.Background())
	res, err := sess.RegisterChatGPT(context.Background(), "a@b.com", func(ctx context.Context) (string, error) {
		return "000000", nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Email != "a@b.com" || res.Meta["dry_run"] != true {
		t.Fatalf("%+v", res)
	}
	if res.Meta["proxy"] == "http://user:secret@1.2.3.4:8080" {
		t.Fatal("proxy should be redacted in meta")
	}
}

func TestLiveMissingBinary(t *testing.T) {
	e := &Engine{DryRun: false, Binary: "/nonexistent/cloak-binary"}
	if e.Available() {
		t.Fatal("missing binary must not be available")
	}
	_, err := e.Open(context.Background(), domain.BrowserOpenOptions{})
	if err == nil {
		t.Fatal("expected error")
	}
}
