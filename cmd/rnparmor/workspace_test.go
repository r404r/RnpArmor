package main

import "testing"

func TestResolveStartupWorkspace(t *testing.T) {
	const home = "/Users/me"
	cases := []struct {
		wd, home, want string
	}{
		{wd: "/Users/me/keys", home: home, want: "/Users/me/keys"},
		{wd: "/home/me/keys", home: home, want: "/home/me/keys"},
		{wd: `C:\Users\me\keys`, home: home, want: `C:\Users\me\keys`},
		{wd: "/", home: home, want: home},
		{wd: "", home: home, want: home},
		{wd: ".", home: home, want: home},
		{wd: `C:\`, home: home, want: home},
		{wd: "C:/", home: home, want: home},
		{wd: "C:", home: home, want: home},
		{wd: "/", home: "", want: ""},
		{wd: "", home: "", want: ""},
		{wd: ".", home: "", want: ""},
		{wd: `C:\`, home: "", want: ""},
		{wd: "C:/", home: "", want: ""},
	}
	for _, tc := range cases {
		got := resolveStartupWorkspace(tc.wd, tc.home)
		if got != tc.want {
			t.Errorf("resolveStartupWorkspace(%q, %q) = %q, want %q", tc.wd, tc.home, got, tc.want)
		}
	}
}
