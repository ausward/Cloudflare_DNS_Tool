package get_config

import (
	"os"
	"testing"
)

// ----------- Test Match_string -------------------

func TestMatchString(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		target  string
		want    bool
		wantErr bool
	}{
		// Exact match
		{"Exact match", `^example\.com$`, "example.com", true, false},
		{"No match (subdomain not allowed)", `^example\.com$`, "sub.example.com", false, false},

		// Subdomain wildcard
		{"Wildcard subdomain", `^.*\.example\.com$`, "dev.example.com", true, false},
		{"Wildcard match nested subdomain", `^.*\.example\.com$`, "api.dev.example.com", true, false},
		{"Wildcard fails on root", `^.*\.example\.com$`, "example.com", false, false},

		// Allow both root and subdomains
		{"Root or subdomain match", `(^|\.)example\.com$`, "example.com", true, false},
		{"Subdomain match via root or sub", `(^|\.)example\.com$`, "test.example.com", true, false},
		{"Non-match with different domain", `(^|\.)example\.com$`, "badexample.com", false, false},

		// Invalid regex
		{"Invalid regex", `[`, "example.com", false, true},
		{"sub sub but not sub", `^.+\.reports\.example\.us$`, "ninjio.reports.example.us", true, false },
		{"sub sub but not sub", `^.+\.reports\.example\.us$`, "reports.example.us", false, false },
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Match_string(tt.pattern, tt.target)
			if (err != nil) != tt.wantErr {
				t.Errorf("Match_string(%q, %q) error = %v, wantErr %v", tt.pattern, tt.target, err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Match_string(%q, %q) = %v, want %v", tt.pattern, tt.target, got, tt.want)
			}
		})
	}
}

// ----------- Test Read_yaml ----------------------

func TestReadYaml(t *testing.T) {
	mock := `
content: 1.2.3.4
name: test.example.com
type: A
proxied: false
comment: Test record
tags: ["test"]
ttl: 120
`
	_ = os.MkdirAll("./CONFIG", 0755)
	err := os.WriteFile("./CONFIG/create.yaml", []byte(mock), 0644)
	if err != nil {
		t.Fatalf("Failed to write test yaml: %v", err)
	}
	defer os.Remove("./CONFIG/create.yaml")

	c, err := Read_yaml()
	if err != nil {
		t.Fatalf("Read_yaml() failed: %v", err)
	}

	if c.Name != "test.example.com" {
		t.Errorf("Expected name 'test.example.com', got '%s'", c.Name)
	}
	if c.Content != "1.2.3.4" {
		t.Errorf("Expected content '1.2.3.4', got '%s'", c.Content)
	}
}

// ----------- Test Read_ignore --------------------

func TestReadIgnore(t *testing.T) {
	mock := `
ignore:
  - domain: example.com
    desired_ip: 1.2.3.4
`
	_ = os.MkdirAll("./CONFIG", 0755)
	err := os.WriteFile("./CONFIG/ignore.yaml", []byte(mock), 0644)
	if err != nil {
		t.Fatalf("Failed to write ignore.yaml: %v", err)
	}
	defer os.Remove("./CONFIG/ignore.yaml")

	ig, err := Read_ignore()
	if err != nil {
		t.Fatalf("Read_ignore() failed: %v", err)
	}
	if len(ig.Ignore) != 1 || ig.Ignore[0].Domain != "example.com" {
		t.Errorf("Expected ignore domain 'example.com', got %+v", ig.Ignore)
	}
}

// ----------- Test Get_account_info ---------------

func TestGetAccountInfo(t *testing.T) {
	mock := `
X-Auth-Email: test@example.com
X-Auth-Key: secret123
`
	_ = os.MkdirAll("./CONFIG", 0755)
	err := os.WriteFile("./CONFIG/config.yaml", []byte(mock), 0644)
	if err != nil {
		t.Fatalf("Failed to write config.yaml: %v", err)
	}
	defer os.Remove("./CONFIG/config.yaml")

	email, token := Get_account_info()
	if email != "test@example.com" {
		t.Errorf("Expected email 'test@example.com', got '%s'", email)
	}
	if token != "secret123" {
		t.Errorf("Expected token 'secret123', got '%s'", token)
	}
}
