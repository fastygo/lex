package adapters

import "testing"

func TestSelectRefusesSilentFallback(t *testing.T) {
	cases := []struct {
		name     string
		selector string
		direct   bool
		hosted   bool
		want     Kind
		wantErr  bool
	}{
		{name: "direct only", direct: true, want: KindDirect},
		{name: "hosted only", hosted: true, want: KindHosted},
		{name: "neither", want: KindNone},
		{name: "both without selector", direct: true, hosted: true, wantErr: true},
		{name: "explicit direct", selector: "direct", direct: true, hosted: true, want: KindDirect},
		{name: "explicit hosted", selector: "hosted", direct: true, hosted: true, want: KindHosted},
		{name: "direct missing key", selector: "direct", wantErr: true},
		{name: "hosted missing key", selector: "hosted", wantErr: true},
		{name: "unknown", selector: "latest", direct: true, wantErr: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Select(tc.selector, tc.direct, tc.hosted)
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if got != tc.want {
				t.Fatalf("kind = %q, want %q", got, tc.want)
			}
		})
	}
}
