package mcpserver

// newPagination derives top-level page metadata from the absolute result offset,
// the page size, the total (unfiltered) hit count, and the next-page cursor.
func newPagination(offset, pageSize, totalHits int, nextCursor *string) Pagination {
	p := Pagination{
		PageSize:   pageSize,
		TotalHits:  totalHits,
		NextCursor: nextCursor,
		HasMore:    nextCursor != nil,
	}
	if pageSize > 0 {
		p.Page = offset/pageSize + 1
		p.TotalPages = (totalHits + pageSize - 1) / pageSize
	} else {
		p.Page = 1
		if totalHits > 0 {
			p.TotalPages = 1
		}
	}
	return p
}
