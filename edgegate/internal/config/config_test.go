package config

import "testing"

func TestValidate(t *testing.T) {
	cases := []struct {
		name string
		cfg  Config
		want bool
	}{
		{"valid", Config{Routes: []Route{{Path: "/api", Backends: []string{"http://localhost:8081"}}}}, true},
		{"missing path", Config{Routes: []Route{{Backends: []string{"http://localhost:8081"}}}}, false},
		{"invalid backend", Config{Routes: []Route{{Path: "/api", Backends: []string{"localhost:8081"}}}}, false},
		{"invalid strategy", Config{Routes: []Route{{Path: "/api", Backends: []string{"http://localhost:8081"}, Strategy: "random"}}}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := Validate(tc.cfg) == nil; got != tc.want {
				t.Fatalf("valid=%v, want %v", got, tc.want)
			}
		})
	}
}
