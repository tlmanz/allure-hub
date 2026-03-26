package allure

import "testing"

func TestShouldIgnoreAllureParseErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		stderr string
		want   bool
	}{
		{
			name:   "known testrun parse error",
			stderr: "error parsing testrun.json TypeError: parsed is not iterable",
			want:   true,
		},
		{
			name:   "known history parse error",
			stderr: "error parsing history.json TypeError: parsed is not iterable",
			want:   true,
		},
		{
			name: "both known parse errors",
			stderr: "error parsing testrun.json TypeError: parsed is not iterable\n" +
				"error parsing history.json TypeError: parsed is not iterable",
			want: true,
		},
		{
			name: "known parse error mixed with unknown stderr",
			stderr: "error parsing testrun.json TypeError: parsed is not iterable\n" +
				"fatal: something else broke",
			want: false,
		},
		{
			name:   "unknown parse error shape",
			stderr: "error parsing testrun.json unexpected token",
			want:   false,
		},
		{
			name:   "empty stderr",
			stderr: "",
			want:   false,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := shouldIgnoreAllureParseErrors(tc.stderr)
			if got != tc.want {
				t.Fatalf("shouldIgnoreAllureParseErrors() = %v, want %v", got, tc.want)
			}
		})
	}
}
