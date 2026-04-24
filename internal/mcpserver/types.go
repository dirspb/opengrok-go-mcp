package mcpserver

type SearchCodeInput struct {
	Project      string   `json:"project,omitempty" jsonschema:"optional OpenGrok project override; omit unless the user explicitly names an OpenGrok project"`
	Projects     []string `json:"projects,omitempty" jsonschema:"optional OpenGrok project overrides; omit unless the user explicitly names OpenGrok projects"`
	Query        string   `json:"query"`
	Mode         string   `json:"mode,omitempty"`
	PathPrefix   string   `json:"path_prefix"`
	FileType     string   `json:"file_type"`
	PageSize     int      `json:"page_size"`
	Cursor       *string  `json:"cursor,omitempty"`
	IncludeLinks *bool    `json:"include_links,omitempty"`
	ExpandContext *bool   `json:"expand_context,omitempty"`
}

type SymbolSearchInput struct {
	Project      string   `json:"project,omitempty" jsonschema:"optional OpenGrok project override; omit unless the user explicitly names an OpenGrok project"`
	Projects     []string `json:"projects,omitempty" jsonschema:"optional OpenGrok project overrides; omit unless the user explicitly names OpenGrok projects"`
	Symbol       string   `json:"symbol"`
	PageSize     int      `json:"page_size"`
	Cursor       *string  `json:"cursor,omitempty"`
	IncludeLinks *bool    `json:"include_links,omitempty"`
	ExpandContext *bool   `json:"expand_context,omitempty"`
}

type SearchOutput struct {
	Project     string      `json:"project"`
	Mode        string      `json:"mode"`
	Query       string      `json:"query"`
	TotalHits   int         `json:"total_hits"`
	Results     []Result    `json:"results"`
	PageSize    int         `json:"page_size"`
	NextCursor  *string     `json:"next_cursor"`
	Warning     *string     `json:"warning,omitempty"`
	Diagnostics Diagnostics `json:"diagnostics"`
}

type Diagnostics struct {
	OffsetUsed         int `json:"offset_used"`
	OpenGrokStart      int `json:"opengrok_start"`
	OpenGrokMaxResults int `json:"opengrok_max_results"`
}

type Result struct {
	ResultID     string         `json:"result_id"`
	Project      string         `json:"project"`
	FilePath     string         `json:"file_path"`
	LineNumber   int            `json:"line_number"`
	ColumnNumber *int           `json:"column_number"`
	Kind         string         `json:"kind"`
	Symbol       *string        `json:"symbol"`
	Snippet      string         `json:"snippet"`
	DisplayTitle string         `json:"display_title"`
	DisplayURL   string         `json:"display_url"`
	RawURL       *string        `json:"raw_url"`
	Citation     Citation       `json:"citation"`
	ResourceURI  string         `json:"resource_uri"`
	Score        *float64       `json:"score"`
	Context      *ResultContext `json:"context,omitempty"`
	Metadata     map[string]any `json:"metadata"`
}

type Citation struct {
	Title string `json:"title"`
	URL   string `json:"url"`
	Line  int    `json:"line,omitempty"`
}

type ResultContext struct {
	Content   string `json:"content"`
	StartLine int    `json:"start_line"`
	EndLine   int    `json:"end_line"`
}

type ListProjectsInput struct {
	Cursor *string `json:"cursor,omitempty"`
}

type ProjectItem struct {
	Project     string `json:"project"`
	Title       string `json:"title"`
	Description string `json:"description"`
	ProjectURL  string `json:"project_url"`
	ResourceURI string `json:"resource_uri"`
}

type ListProjectsOutput struct {
	Projects      []ProjectItem `json:"projects"`
	TotalProjects int           `json:"total_projects"`
	NextCursor    *string       `json:"next_cursor"`
}

type FileContextInput struct {
	Project            string  `json:"project,omitempty" jsonschema:"optional OpenGrok project override; omit unless the user explicitly names an OpenGrok project"`
	FilePath           string  `json:"file_path" jsonschema:"project-relative file path"`
	LineNumber         int     `json:"line_number,omitempty"`
	Before             int     `json:"before,omitempty"`
	After              int     `json:"after,omitempty"`
	IncludeAnnotations bool    `json:"include_annotations,omitempty"`
	IncludeLinks       *bool   `json:"include_links,omitempty"`
	Cursor             *string `json:"cursor,omitempty"`
}

type FileContextOutput struct {
	Project              string   `json:"project"`
	FilePath             string   `json:"file_path"`
	LineNumber           int      `json:"line_number"`
	StartLine            int      `json:"start_line"`
	EndLine              int      `json:"end_line"`
	TotalLines           int      `json:"total_lines"`
	Truncated            bool     `json:"truncated"`
	Content              string   `json:"content"`
	DisplayURL           string   `json:"display_url"`
	RawURL               *string  `json:"raw_url"`
	Citation             Citation `json:"citation"`
	NextCursor           *string  `json:"next_cursor,omitempty"`
	Hint                 *string  `json:"hint,omitempty"`
	AnnotationsAvailable bool     `json:"annotations_available"`
	ResourceURI          string   `json:"resource_uri"`
}
