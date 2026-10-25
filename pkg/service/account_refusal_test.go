package service

import "testing"

func TestRefusal(t *testing.T) {
	cases := []struct {
		status int
		body   string
		want   string
	}{
		{403, `{"message":"this role setup wasn't issued to you, or it has expired or been used: run the registration again"}`,
			"this role setup wasn't issued to you, or it has expired or been used: run the registration again"},
		{500, `{"error":"failed to build role config"}`, `status 500: {"error":"failed to build role config"}`},
		{502, ``, "status 502"},
	}
	for _, c := range cases {
		if got := refusal(c.status, []byte(c.body)).Error(); got != c.want {
			t.Errorf("refusal(%d, %q) = %q, want %q", c.status, c.body, got, c.want)
		}
	}
}
