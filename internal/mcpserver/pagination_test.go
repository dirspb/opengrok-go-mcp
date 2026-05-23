package mcpserver

import "testing"

func TestNewPagination(t *testing.T) {
	cur := "next"
	cases := []struct {
		name                        string
		offset, pageSize, totalHits int
		nextCursor                  *string
		wantPage, wantTotalPages    int
		wantHasMore                 bool
	}{
		{"empty", 0, 20, 0, nil, 1, 0, false},
		{"partial single page", 0, 20, 5, nil, 1, 1, false},
		{"exact single page", 0, 20, 20, nil, 1, 1, false},
		{"first of three", 0, 20, 45, &cur, 1, 3, true},
		{"middle of three", 20, 20, 45, &cur, 2, 3, true},
		{"last of three", 40, 20, 45, nil, 3, 3, false},
		{"exact multiple two pages", 0, 20, 40, &cur, 1, 2, true},
		{"zero page size guarded", 0, 0, 10, nil, 1, 1, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := newPagination(tc.offset, tc.pageSize, tc.totalHits, tc.nextCursor)
			if p.Page != tc.wantPage {
				t.Errorf("Page = %d, want %d", p.Page, tc.wantPage)
			}
			if p.TotalPages != tc.wantTotalPages {
				t.Errorf("TotalPages = %d, want %d", p.TotalPages, tc.wantTotalPages)
			}
			if p.HasMore != tc.wantHasMore {
				t.Errorf("HasMore = %v, want %v", p.HasMore, tc.wantHasMore)
			}
			if p.TotalHits != tc.totalHits {
				t.Errorf("TotalHits = %d, want %d", p.TotalHits, tc.totalHits)
			}
			if p.PageSize != tc.pageSize {
				t.Errorf("PageSize = %d, want %d", p.PageSize, tc.pageSize)
			}
		})
	}
}
