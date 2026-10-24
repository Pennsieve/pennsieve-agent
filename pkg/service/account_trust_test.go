package service

import (
	"encoding/json"
	"testing"
)

func TestRequireExternalId(t *testing.T) {
	legacy := `{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Principal":{"AWS":"111122223333"},"Action":"sts:AssumeRole"}]}`
	out, err := requireExternalId(legacy, "pn-0123456789abcdef0123456789abcdef")
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		Statement []struct {
			Principal map[string]string
			Condition map[string]map[string]string
		}
	}
	if err := json.Unmarshal([]byte(out), &doc); err != nil {
		t.Fatal(err)
	}
	if doc.Statement[0].Principal["AWS"] != "111122223333" {
		t.Fatalf("principal lost: %s", out)
	}
	if doc.Statement[0].Condition["StringEquals"]["sts:ExternalId"] != "pn-0123456789abcdef0123456789abcdef" {
		t.Fatalf("condition missing: %s", out)
	}
	if _, err := requireExternalId(`{"Version":"2012-10-17"}`, "x"); err == nil {
		t.Fatal("a policy without statements should fail")
	}
}
