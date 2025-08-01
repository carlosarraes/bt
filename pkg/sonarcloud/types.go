package sonarcloud

import "time"

type ComponentMeasure struct {
	Component struct {
		ID        string `json:"id"`
		Key       string `json:"key"`
		Name      string `json:"name"`
		Qualifier string `json:"qualifier"`
		Measures  []struct {
			Metric    string `json:"metric"`
			Value     string `json:"value"`
			BestValue bool   `json:"bestValue"`
			Periods   []struct {
				Index     int    `json:"index"`
				Value     string `json:"value"`
				BestValue bool   `json:"bestValue"`
			} `json:"periods,omitempty"`
		} `json:"measures"`
	} `json:"component"`
}

type ComponentTree struct {
	Paging struct {
		PageIndex int `json:"pageIndex"`
		PageSize  int `json:"pageSize"`
		Total     int `json:"total"`
	} `json:"paging"`
	BaseComponent struct {
		ID        string `json:"id"`
		Key       string `json:"key"`
		Name      string `json:"name"`
		Qualifier string `json:"qualifier"`
	} `json:"baseComponent"`
	Components []ComponentTreeItem `json:"components"`
}

type ComponentTreeItem struct {
	ID        string `json:"id"`
	Key       string `json:"key"`
	Name      string `json:"name"`
	Qualifier string `json:"qualifier"`
	Path      string `json:"path"`
	Language  string `json:"language"`
	Measures  []struct {
		Metric    string `json:"metric"`
		Value     string `json:"value"`
		BestValue bool   `json:"bestValue"`
		Periods   []struct {
			Index     int    `json:"index"`
			Value     string `json:"value"`
			BestValue bool   `json:"bestValue"`
		} `json:"periods,omitempty"`
	} `json:"measures"`
}

type QualityGate struct {
	ProjectStatus struct {
		Status     string `json:"status"`
		Conditions []struct {
			Status         string `json:"status"`
			MetricKey      string `json:"metricKey"`
			Comparator     string `json:"comparator"`
			PeriodIndex    int    `json:"periodIndex,omitempty"`
			ErrorThreshold string `json:"errorThreshold,omitempty"`
			ActualValue    string `json:"actualValue,omitempty"`
		} `json:"conditions"`
		Periods []struct {
			Index int       `json:"index"`
			Mode  string    `json:"mode"`
			Date  time.Time `json:"date"`
		} `json:"periods"`
	} `json:"projectStatus"`
}

type IssuesSearch struct {
	Total   int     `json:"total"`
	P       int     `json:"p"`
	PS      int     `json:"ps"`
	Paging  Paging  `json:"paging"`
	Issues  []Issue `json:"issues"`
	Rules   []Rule  `json:"rules"`
	Facets  []Facet `json:"facets"`
}

type Issue struct {
	Key           string            `json:"key"`
	Rule          string            `json:"rule"`
	Severity      string            `json:"severity"`
	Component     string            `json:"component"`
	Project       string            `json:"project"`
	Line          *int              `json:"line,omitempty"`
	Hash          string            `json:"hash"`
	TextRange     *TextRange        `json:"textRange,omitempty"`
	Flows         []Flow            `json:"flows"`
	Resolution    string            `json:"resolution,omitempty"`
	Status        string            `json:"status"`
	Message       string            `json:"message"`
	Effort        string            `json:"effort,omitempty"`
	Debt          string            `json:"debt,omitempty"`
	Author        string            `json:"author,omitempty"`
	Tags          []string          `json:"tags"`
	Transitions   []string          `json:"transitions"`
	Actions       []string          `json:"actions"`
	Comments      []Comment         `json:"comments"`
	CreationDate  time.Time         `json:"creationDate"`
	UpdateDate    time.Time         `json:"updateDate"`
	CloseDate     *time.Time        `json:"closeDate,omitempty"`
	Type          string            `json:"type"`
	Scope         string            `json:"scope,omitempty"`
	QuickFixAvailable bool          `json:"quickFixAvailable,omitempty"`
	MessageFormattings []MessageFormatting `json:"messageFormattings,omitempty"`
}

type TextRange struct {
	StartLine   int `json:"startLine"`
	EndLine     int `json:"endLine"`
	StartOffset int `json:"startOffset"`
	EndOffset   int `json:"endOffset"`
}

type Flow struct {
	Locations []Location `json:"locations"`
}

type Location struct {
	Component   string     `json:"component"`
	TextRange   *TextRange `json:"textRange,omitempty"`
	Message     string     `json:"msg"`
	MessageFormattings []MessageFormatting `json:"msgFormattings,omitempty"`
}

type MessageFormatting struct {
	Start int    `json:"start"`
	End   int    `json:"end"`
	Type  string `json:"type"`
}

type Comment struct {
	Key           string    `json:"key"`
	Login         string    `json:"login"`
	HTMLText      string    `json:"htmlText"`
	Markdown      string    `json:"markdown"`
	Updatable     bool      `json:"updatable"`
	CreatedAt     time.Time `json:"createdAt"`
}

type Rule struct {
	Key  string `json:"key"`
	Name string `json:"name"`
	Lang string `json:"lang"`
}

type Facet struct {
	Property string       `json:"property"`
	Values   []FacetValue `json:"values"`
}

type FacetValue struct {
	Val   string `json:"val"`
	Count int    `json:"count"`
}

type Paging struct {
	PageIndex int `json:"pageIndex"`
	PageSize  int `json:"pageSize"`
	Total     int `json:"total"`
}

type SourceLines struct {
	Sources []SourceLine `json:"sources"`
}

type SourceLine struct {
	Line                int     `json:"line"`
	Code                string  `json:"code"`
	SCMRevision         string  `json:"scmRevision,omitempty"`
	SCMAuthor           string  `json:"scmAuthor,omitempty"`
	SCMDate             *string `json:"scmDate,omitempty"`
	UTLineHits          *int    `json:"utLineHits,omitempty"`
	UTConditions        *int    `json:"utConditions,omitempty"`
	UTCoveredConditions *int    `json:"utCoveredConditions,omitempty"`
	ITLineHits          *int    `json:"itLineHits,omitempty"`
	ITConditions        *int    `json:"itConditions,omitempty"`
	ITCoveredConditions *int    `json:"itCoveredConditions,omitempty"`
	LineHits            *int    `json:"lineHits,omitempty"`
	Conditions          *int    `json:"conditions,omitempty"`
	CoveredConditions   *int    `json:"coveredConditions,omitempty"`
	Duplicated          bool    `json:"duplicated"`
	IsNew               bool    `json:"isNew"`
}

type Report struct {
	ProjectKey    string           `json:"projectKey"`
	PullRequestID *int             `json:"pullRequestId,omitempty"`
	Timestamp     time.Time        `json:"timestamp"`
	QualityGate   *QualityGateInfo `json:"qualityGate,omitempty"`
	Coverage      *CoverageData    `json:"coverage,omitempty"`
	Issues        *IssuesData      `json:"issues,omitempty"`
	Metrics       *MetricsData     `json:"metrics,omitempty"`
	Warnings      []error          `json:"warnings,omitempty"`
}

type QualityGateInfo struct {
	Status          string                    `json:"status"`
	Passed          bool                      `json:"passed"`
	Conditions      []QualityGateCondition    `json:"conditions"`
	FailedConditions []QualityGateCondition   `json:"failedConditions"`
	Error           string                    `json:"error,omitempty"`
}

type QualityGateCondition struct {
	MetricKey       string `json:"metricKey"`
	MetricName      string `json:"metricName"`
	Comparator      string `json:"comparator"`
	Threshold       string `json:"threshold"`
	ActualValue     string `json:"actualValue"`
	Status          string `json:"status"`
	Failed          bool   `json:"failed"`
	OnNewCode       bool   `json:"onNewCode"`
}

type CoverageData struct {
	Available         bool                `json:"available"`
	OverallCoverage   float64             `json:"overallCoverage"`
	NewCodeCoverage   float64             `json:"newCodeCoverage"`
	Files             []CoverageFile      `json:"files"`
	UncoveredLines    []UncoveredLine     `json:"uncoveredLines"`
	CoverageDetails   []CoverageDetails   `json:"coverageDetails"`
	Summary           CoverageSummary     `json:"summary"`
	Error             string              `json:"error,omitempty"`
}

type CoverageFile struct {
	Path            string  `json:"path"`
	Name            string  `json:"name"`
	Coverage        float64 `json:"coverage"`
	NewCoverage     float64 `json:"newCoverage"`
	UncoveredLines  int     `json:"uncoveredLines"`
	NewUncoveredLines int   `json:"newUncoveredLines"`
	Language        string  `json:"language"`
	ComponentKey    string  `json:"componentKey"`
}

type UncoveredLine struct {
	File       string `json:"file"`
	Line       int    `json:"line"`
	Code       string `json:"code"`
	IsNew      bool   `json:"isNew"`
}

type CoverageDetails struct {
	FilePath        string          `json:"filePath"`
	FileName        string          `json:"fileName"`
	CoveragePercent float64         `json:"coveragePercent"`
	TotalUncovered  int             `json:"totalUncovered"`
	UncoveredLines  []UncoveredLine `json:"uncoveredLines"`
	NewUncovered    int             `json:"newUncovered"`
	Language        string          `json:"language"`
}

type CoverageSummary struct {
	TotalLines        int `json:"totalLines"`
	CoveredLines      int `json:"coveredLines"`
	UncoveredLines    int `json:"uncoveredLines"`
	NewTotalLines     int `json:"newTotalLines"`
	NewCoveredLines   int `json:"newCoveredLines"`
	NewUncoveredLines int `json:"newUncoveredLines"`
}

type IssuesData struct {
	Available        bool               `json:"available"`
	TotalIssues      int                `json:"totalIssues"`
	NewIssues        int                `json:"newIssues"`
	Bugs             int                `json:"bugs"`
	Vulnerabilities  int                `json:"vulnerabilities"`
	CodeSmells       int                `json:"codeSmells"`
	SecurityHotspots int                `json:"securityHotspots"`
	Issues           []ProcessedIssue   `json:"issues"`
	Summary          IssuesSummary      `json:"summary"`
	Error            string             `json:"error,omitempty"`
}

type ProcessedIssue struct {
	Key          string    `json:"key"`
	Type         string    `json:"type"`
	Severity     string    `json:"severity"`
	Rule         string    `json:"rule"`
	RuleName     string    `json:"ruleName"`
	Component    string    `json:"component"`
	File         string    `json:"file"`
	Line         *int      `json:"line,omitempty"`
	Message      string    `json:"message"`
	Effort       string    `json:"effort,omitempty"`
	TechnicalDebt string   `json:"technicalDebt,omitempty"`
	IsNew        bool      `json:"isNew"`
	CreatedAt    time.Time `json:"createdAt"`
}

type IssuesSummary struct {
	BySeverity map[string]int `json:"bySeverity"`
	ByType     map[string]int `json:"byType"`
	ByLanguage map[string]int `json:"byLanguage"`
	TechnicalDebt time.Duration `json:"technicalDebt"`
}

type MetricsData struct {
	Available     bool               `json:"available"`
	Metrics       map[string]string  `json:"metrics"`
	Ratings       map[string]string  `json:"ratings"`
	Duplication   float64            `json:"duplication"`
	Error         string             `json:"error,omitempty"`
}

type PaginationStrategy struct {
	MaxPages    int  `json:"maxPages"`
	MaxResults  int  `json:"maxResults"`
	PageSize    int  `json:"pageSize"`
	EarlyExit   bool `json:"earlyExit"`
}

type FilterOptions struct {
	IncludeCoverage      bool     `json:"includeCoverage"`
	IncludeIssues        bool     `json:"includeIssues"`
	CoverageThreshold    float64  `json:"coverageThreshold"`
	Limit                int      `json:"limit"`
	NewCodeOnly          bool     `json:"newCodeOnly"`
	SeverityFilter       []string `json:"severityFilter"`
	ShowWorstFirst       bool     `json:"showWorstFirst"`
	ShowAllLines         bool     `json:"showAllLines"`
	LinesPerFile         int      `json:"linesPerFile"`
	NewLinesOnly         bool     `json:"newLinesOnly"`
	MinUncoveredLines    int      `json:"minUncoveredLines"`
	MaxUncoveredLines    int      `json:"maxUncoveredLines"`
	FilePattern          string   `json:"filePattern"`
	NoLineDetails        bool     `json:"noLineDetails"`
	TruncateLines        int      `json:"truncateLines"`
	Debug                bool     `json:"debug"`
}
