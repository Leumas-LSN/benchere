package report

import "testing"

func TestComputeCounts(t *testing.T) {
	cases := []struct {
		name string
		in   []ProfileSummary
		want JobCounts
	}{
		{"empty", nil, JobCounts{}},
		{"all pass", []ProfileSummary{{Verdict: "pass"}, {Verdict: "pass"}}, JobCounts{Total: 2, Pass: 2}},
		{"mixed", []ProfileSummary{{Verdict: "pass"}, {Verdict: "fail"}, {Verdict: "n/a"}}, JobCounts{Total: 3, Pass: 1, Fail: 1, NA: 1}},
		{"unknown verdict treated as n/a", []ProfileSummary{{Verdict: ""}, {Verdict: "weird"}}, JobCounts{Total: 2, NA: 2}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := computeCounts(tc.in)
			if got != tc.want {
				t.Fatalf("got %+v, want %+v", got, tc.want)
			}
		})
	}
}
