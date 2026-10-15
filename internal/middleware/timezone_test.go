package middleware

import "testing"

func TestResolveLocation(t *testing.T) {
	cases := map[string]string{
		"Europe%2FIstanbul":   "Europe/Istanbul", // as set by encodeURIComponent
		"America/New_York":    "America/New_York",
		"Etc%2FGMT%2B3":       "Etc/GMT+3", // '+' must survive decoding
		"":                    DefaultTimezone,
		"Not/AZone":           DefaultTimezone,
		"Local":               DefaultTimezone,
		"%zz":                 DefaultTimezone,
		"../../../etc/passwd": DefaultTimezone,
	}
	for raw, want := range cases {
		if got := ResolveLocation(raw).String(); got != want {
			t.Errorf("ResolveLocation(%q) = %s, want %s", raw, got, want)
		}
	}
}
