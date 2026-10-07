package service

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/kirban/social-media/internal/model"
)

// window builds a cached page the way warmCachedPage does: oldest first, so
// entry i holds message "m<i>" and the newest message is the last element.
func window(t *testing.T, n int) [][]byte {
	t.Helper()
	entries := make([][]byte, n)
	for i := range n {
		payload, err := json.Marshal(model.DialogMessage{Text: fmt.Sprintf("m%d", i)})
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		entries[i] = payload
	}
	return entries
}

func texts(messages []model.DialogMessage) []string {
	out := make([]string, len(messages))
	for i, m := range messages {
		out[i] = m.Text
	}
	return out
}

func TestDecodePage(t *testing.T) {
	tests := []struct {
		name   string
		size   int
		limit  int64
		offset int64
		wantOK bool
		want   []string
	}{
		{
			name:   "newest page first",
			size:   5,
			limit:  2,
			offset: 0,
			wantOK: true,
			want:   []string{"m4", "m3"},
		},
		{
			name:   "second page walks backwards",
			size:   5,
			limit:  2,
			offset: 2,
			wantOK: true,
			want:   []string{"m2", "m1"},
		},
		{
			name:   "page ending exactly at the oldest entry",
			size:   5,
			limit:  5,
			offset: 0,
			wantOK: true,
			want:   []string{"m4", "m3", "m2", "m1", "m0"},
		},
		{
			name:   "last entry alone",
			size:   5,
			limit:  1,
			offset: 4,
			wantOK: true,
			want:   []string{"m0"},
		},
		{
			// Straddling the end of the window must be refused rather than
			// answered short: a short page looks like the end of the history.
			name:   "page reaching past the window is refused",
			size:   5,
			limit:  3,
			offset: 3,
			wantOK: false,
		},
		{
			name:   "offset at the window edge is refused",
			size:   5,
			limit:  1,
			offset: 5,
			wantOK: false,
		},
		{
			name:   "empty window is refused",
			size:   0,
			limit:  1,
			offset: 0,
			wantOK: false,
		},
		{
			name:   "non-positive limit is refused",
			size:   5,
			limit:  0,
			offset: 0,
			wantOK: false,
		},
		{
			name:   "negative offset is refused",
			size:   5,
			limit:  1,
			offset: -1,
			wantOK: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := decodePage(window(t, tc.size), tc.limit, tc.offset)
			if ok != tc.wantOK {
				t.Fatalf("decodePage ok = %v, want %v", ok, tc.wantOK)
			}
			if !tc.wantOK {
				return
			}
			if diff := texts(got); !equal(diff, tc.want) {
				t.Fatalf("decodePage = %v, want %v", diff, tc.want)
			}
		})
	}
}

// A corrupt entry must be reported as decodable-but-broken (ok, nil) so the
// caller drops the page instead of serving a partial one.
func TestDecodePageCorruptEntry(t *testing.T) {
	entries := window(t, 3)
	entries[1] = []byte("not json")

	got, ok := decodePage(entries, 3, 0)
	if !ok {
		t.Fatal("decodePage reported the window as not covering the page")
	}
	if got != nil {
		t.Fatalf("decodePage = %v, want nil for a corrupt entry", texts(got))
	}
}

func TestPageOf(t *testing.T) {
	all := []model.DialogMessage{{Text: "a"}, {Text: "b"}, {Text: "c"}}

	tests := []struct {
		name          string
		limit, offset int64
		want          []string
	}{
		{"full page", 3, 0, []string{"a", "b", "c"}},
		{"first page", 2, 0, []string{"a", "b"}},
		{"middle", 1, 1, []string{"b"}},
		{"limit past the end is clamped", 10, 1, []string{"b", "c"}},
		{"offset past the end is empty", 2, 3, []string{}},
		{"offset far past the end is empty", 2, 99, []string{}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := texts(pageOf(all, tc.limit, tc.offset)); !equal(got, tc.want) {
				t.Fatalf("pageOf = %v, want %v", got, tc.want)
			}
		})
	}
}

func equal(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
