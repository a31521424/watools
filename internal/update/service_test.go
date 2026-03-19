package update

import "testing"

func TestCompareVersions(t *testing.T) {
	testCases := []struct {
		name    string
		next    string
		current string
		want    int
		wantErr bool
	}{
		{name: "higher patch", next: "1.0.1", current: "1.0.0", want: 1},
		{name: "same version", next: "1.0.0", current: "1.0.0", want: 0},
		{name: "lower version", next: "0.9.9", current: "1.0.0", want: -1},
		{name: "prefixed version", next: "v1.2.0", current: "1.1.9", want: 1},
		{name: "release beats prerelease", next: "1.0.0", current: "1.0.0-beta.1", want: 1},
		{name: "prerelease ordering", next: "1.0.0-beta.2", current: "1.0.0-beta.1", want: 1},
		{name: "numeric prerelease lower than text", next: "1.0.0-1", current: "1.0.0-beta", want: -1},
		{name: "invalid version", next: "broken", current: "1.0.0", wantErr: true},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			got, err := compareVersions(testCase.next, testCase.current)
			if testCase.wantErr {
				if err == nil {
					t.Fatal("expected compareVersions to return an error")
				}
				return
			}
			if err != nil {
				t.Fatalf("compareVersions returned error: %v", err)
			}
			if got != testCase.want {
				t.Fatalf("compareVersions(%q, %q) = %d, want %d", testCase.next, testCase.current, got, testCase.want)
			}
		})
	}
}
