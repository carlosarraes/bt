package sonarcloud

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Regression: lines with hits>0 but uncovered branches (e.g. `a ?? b`, `a || b`)
// must be reported. Previously the line scan only flagged lines where
// LineHits == 0, so partial-branch coverage was invisible — letting PRs
// fail the SonarCloud quality gate with no diagnostic surface in `bt`.
func TestGetUncoveredLinesForFile_FlagsConditionGaps(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
		  "sources": [
		    {"line": 10, "code": "covered line", "lineHits": 1, "isNew": true},
		    {"line": 20, "code": "const x = a ?? b;", "lineHits": 1, "conditions": 4, "coveredConditions": 3, "isNew": true},
		    {"line": 30, "code": "if (!c || !d?.e) {", "lineHits": 1, "conditions": 7, "coveredConditions": 6, "isNew": true},
		    {"line": 40, "code": "fully covered branch", "lineHits": 1, "conditions": 2, "coveredConditions": 2, "isNew": true},
		    {"line": 50, "code": "untouched line", "lineHits": 0, "isNew": true}
		  ]
		}`))
	}))
	defer srv.Close()

	svc := &Service{client: newTestClient(t, srv.URL)}
	apiCtx := APIContext{ProjectKey: "p", BaseParams: map[string]string{"component": "p"}, IsPullRequest: true, PullRequestID: 1}
	file := CoverageFile{Path: "src/foo.ts", Name: "foo.ts", ComponentKey: "p:src/foo.ts"}

	details, err := svc.getUncoveredLinesForFile(context.Background(), file, apiCtx, FilterOptions{ShowAllLines: true})
	require.NoError(t, err)
	require.NotNil(t, details)

	byLine := make(map[int]UncoveredLine, len(details.UncoveredLines))
	for _, l := range details.UncoveredLines {
		byLine[l.Line] = l
	}

	require.Contains(t, byLine, 20, "line 20 (3/4 branches) must be flagged as uncovered")
	require.Contains(t, byLine, 30, "line 30 (6/7 branches) must be flagged as uncovered")
	require.Contains(t, byLine, 50, "line 50 (lineHits=0) must be flagged as uncovered")
	assert.NotContains(t, byLine, 10, "fully covered line must not be flagged")
	assert.NotContains(t, byLine, 40, "line with all branches covered must not be flagged")

	assert.Equal(t, 4, byLine[20].Conditions, "line 20 must record total branch count")
	assert.Equal(t, 3, byLine[20].CoveredConditions, "line 20 must record covered branch count")
	assert.Equal(t, 7, byLine[30].Conditions)
	assert.Equal(t, 6, byLine[30].CoveredConditions)
}

// Regression: --new-lines-only used to filter out files that had 0 uncovered
// new LINES but >0 uncovered new CONDITIONS, returning an empty report and
// hiding the gate failure cause.
func TestFilterEligibleFiles_KeepsConditionOnlyFiles(t *testing.T) {
	svc := &Service{}

	files := []CoverageFile{
		{Path: "branch-only.ts", NewUncoveredLines: 0, NewUncoveredConditions: 2, UncoveredLines: 10, Coverage: 80},
		{Path: "line-and-branch.ts", NewUncoveredLines: 3, NewUncoveredConditions: 1, UncoveredLines: 5, Coverage: 70},
		{Path: "fully-covered-new.ts", NewUncoveredLines: 0, NewUncoveredConditions: 0, UncoveredLines: 5, Coverage: 95},
	}

	eligible := svc.filterEligibleFiles(files, FilterOptions{NewLinesOnly: true}, APIContext{})

	paths := make([]string, 0, len(eligible))
	for _, f := range eligible {
		paths = append(paths, f.Path)
	}
	assert.Contains(t, paths, "branch-only.ts", "files with only condition gaps must be kept under --new-lines-only")
	assert.Contains(t, paths, "line-and-branch.ts")
	assert.NotContains(t, paths, "fully-covered-new.ts")
}

// Regression: project-level fetch must request and parse condition metrics
// so the report can compute the real new-code denominator (lines + conditions)
// instead of only lines.
func TestFetchAllMeasures_RequestsAndParsesConditionMetrics(t *testing.T) {
	var rawQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rawQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
		  "component": {
		    "id": "id", "key": "p", "name": "p", "qualifier": "TRK",
		    "measures": [
		      {"metric": "new_coverage", "periods": [{"index": 1, "value": "89.5"}]},
		      {"metric": "new_uncovered_lines", "periods": [{"index": 1, "value": "0"}]},
		      {"metric": "new_lines_to_cover", "periods": [{"index": 1, "value": "8"}]},
		      {"metric": "new_conditions_to_cover", "periods": [{"index": 1, "value": "11"}]},
		      {"metric": "new_uncovered_conditions", "periods": [{"index": 1, "value": "2"}]}
		    ]
		  }
		}`))
	}))
	defer srv.Close()

	svc := &Service{client: newTestClient(t, srv.URL)}
	apiCtx := APIContext{ProjectKey: "p", BaseParams: map[string]string{"component": "p"}}

	measures, err := svc.fetchAllMeasures(context.Background(), apiCtx)
	require.NoError(t, err)

	assert.Contains(t, rawQuery, "new_uncovered_conditions", "API request must ask for new_uncovered_conditions")
	assert.Contains(t, rawQuery, "new_conditions_to_cover", "API request must ask for new_conditions_to_cover")

	assert.Equal(t, 2, measures.NewUncoveredConditions, "new_uncovered_conditions must be parsed")
	assert.Equal(t, 11, measures.NewConditionsToCover, "new_conditions_to_cover must be parsed")
}

// Regression: per-file fetch must surface NewUncoveredConditions so the
// file filter and renderer can keep files with branch-only gaps.
func TestGetFileCoverage_ParsesNewUncoveredConditions(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
		  "paging": {"pageIndex": 1, "pageSize": 100, "total": 1},
		  "baseComponent": {"id": "b", "key": "p", "name": "p", "qualifier": "TRK"},
		  "components": [
		    {
		      "id": "f1", "key": "p:src/foo.ts", "name": "foo.ts", "qualifier": "FIL",
		      "path": "src/foo.ts", "language": "ts",
		      "measures": [
		        {"metric": "coverage", "value": "42.6"},
		        {"metric": "uncovered_lines", "value": "414"},
		        {"metric": "new_coverage", "periods": [{"index": 1, "value": "89.5"}]},
		        {"metric": "new_uncovered_lines", "periods": [{"index": 1, "value": "0"}]},
		        {"metric": "new_uncovered_conditions", "periods": [{"index": 1, "value": "2"}]}
		      ]
		    }
		  ]
		}`))
	}))
	defer srv.Close()

	svc := &Service{client: newTestClient(t, srv.URL)}
	apiCtx := APIContext{ProjectKey: "p", BaseParams: map[string]string{"component": "p"}}
	data := &CoverageData{Files: make([]CoverageFile, 0)}

	require.NoError(t, svc.getFileCoverage(context.Background(), apiCtx, data, FilterOptions{}))

	require.Len(t, data.Files, 1)
	assert.Equal(t, 0, data.Files[0].NewUncoveredLines)
	assert.Equal(t, 2, data.Files[0].NewUncoveredConditions, "per-file new_uncovered_conditions must be parsed")
}

// Sanity: getFileCoverage must include condition metrics in its API request.
func TestGetFileCoverage_RequestsConditionMetrics(t *testing.T) {
	var rawQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rawQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"paging":{"pageIndex":1,"pageSize":100,"total":0},"baseComponent":{"id":"b","key":"p","name":"p","qualifier":"TRK"},"components":[]}`))
	}))
	defer srv.Close()

	svc := &Service{client: newTestClient(t, srv.URL)}
	apiCtx := APIContext{ProjectKey: "p", BaseParams: map[string]string{"component": "p"}}
	data := &CoverageData{Files: make([]CoverageFile, 0)}

	require.NoError(t, svc.getFileCoverage(context.Background(), apiCtx, data, FilterOptions{}))

	decoded, err := url.QueryUnescape(rawQuery)
	require.NoError(t, err)
	assert.Contains(t, decoded, "new_uncovered_conditions")
}

// Regression: per-file uncovered details land in goroutine arrival order, so
// the 10-file display cap can hide the highest-impact PR files. Service must
// sort by NewUncovered DESC (TotalUncovered as tiebreaker) before returning.
func TestGetUncoveredLines_SortsByNewUncoveredDesc(t *testing.T) {
	payloads := map[string]string{
		"p:a.go": `{"sources":[{"line":1,"code":"a","lineHits":0,"isNew":true}]}`,
		"p:b.go": `{"sources":[
			{"line":1,"code":"b1","lineHits":0,"isNew":true},
			{"line":2,"code":"b2","lineHits":0,"isNew":true}
		]}`,
		"p:c.go": `{"sources":[{"line":1,"code":"c","lineHits":1}]}`,
		"p:d.go": `{"sources":[
			{"line":1,"code":"d1","lineHits":0,"isNew":true},
			{"line":2,"code":"d2","lineHits":0,"isNew":true},
			{"line":3,"code":"d3","lineHits":0,"isNew":true}
		]}`,
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(payloads[r.URL.Query().Get("key")]))
	}))
	defer srv.Close()

	svc := &Service{client: newTestClient(t, srv.URL)}
	apiCtx := APIContext{
		ProjectKey:    "p",
		BaseParams:    map[string]string{"component": "p"},
		IsPullRequest: true,
		PullRequestID: 1,
	}
	data := &CoverageData{
		Files: []CoverageFile{
			{Path: "a.go", Name: "a.go", ComponentKey: "p:a.go", NewUncoveredLines: 1, UncoveredLines: 1, Coverage: 50},
			{Path: "b.go", Name: "b.go", ComponentKey: "p:b.go", NewUncoveredLines: 2, UncoveredLines: 2, Coverage: 50},
			{Path: "c.go", Name: "c.go", ComponentKey: "p:c.go", NewUncoveredLines: 0, UncoveredLines: 0, Coverage: 100},
			{Path: "d.go", Name: "d.go", ComponentKey: "p:d.go", NewUncoveredLines: 3, UncoveredLines: 3, Coverage: 50},
		},
	}

	require.NoError(t, svc.getUncoveredLines(context.Background(), apiCtx, data, FilterOptions{ShowAllLines: true}))

	got := make([]string, 0, len(data.CoverageDetails))
	for _, d := range data.CoverageDetails {
		got = append(got, d.FilePath)
	}
	assert.Equal(t, []string{"d.go", "b.go", "a.go"}, got,
		"CoverageDetails must be sorted by NewUncovered DESC so the 10-file display cap surfaces worst PR offenders")
}

// Regression: big legacy files with small PR-touched additions used to be
// dropped by the >500 / Min / Max guards because those rules counted overall
// UncoveredLines instead of NEW uncovered. In PR mode, "size" must mean NEW.
func TestFilterEligibleFiles_PRModeUsesNewUncoveredForSizeGuards(t *testing.T) {
	svc := &Service{}

	files := []CoverageFile{
		{Path: "big-legacy.go", UncoveredLines: 600, NewUncoveredLines: 10, Coverage: 30},
	}

	t.Run("non-PR mode drops the file (legacy behavior)", func(t *testing.T) {
		eligible := svc.filterEligibleFiles(files, FilterOptions{}, APIContext{IsPullRequest: false})
		assert.Empty(t, eligible, "non-PR mode keeps the >500 overall-uncovered guard")
	})

	t.Run("PR mode keeps the file", func(t *testing.T) {
		eligible := svc.filterEligibleFiles(files, FilterOptions{}, APIContext{IsPullRequest: true})
		require.Len(t, eligible, 1)
		assert.Equal(t, "big-legacy.go", eligible[0].Path)
	})

	t.Run("PR mode MinUncoveredLines compares to NEW", func(t *testing.T) {
		eligible := svc.filterEligibleFiles(
			files,
			FilterOptions{MinUncoveredLines: 5},
			APIContext{IsPullRequest: true},
		)
		assert.Len(t, eligible, 1, "10 NEW uncovered >= Min=5 -> keep")
	})

	t.Run("PR mode MaxUncoveredLines compares to NEW", func(t *testing.T) {
		eligible := svc.filterEligibleFiles(
			files,
			FilterOptions{MaxUncoveredLines: 5},
			APIContext{IsPullRequest: true},
		)
		assert.Empty(t, eligible, "10 NEW uncovered > Max=5 -> drop")
	})
}
