package api

import (
	"time"
)

// Pipeline represents a Bitbucket pipeline
type Pipeline struct {
	Type                   string                 `json:"type"`
	UUID                   string                 `json:"uuid"`
	BuildNumber            int                    `json:"build_number"`
	Creator                *User                  `json:"creator,omitempty"`
	Repository             *Repository            `json:"repository,omitempty"`
	Target                 *PipelineTarget        `json:"target,omitempty"`
	Trigger                *PipelineTrigger       `json:"trigger,omitempty"`
	State                  *PipelineState         `json:"state,omitempty"`
	Variables              []*PipelineVariable    `json:"variables,omitempty"`
	CreatedOn              *time.Time             `json:"created_on,omitempty"`
	CompletedOn            *time.Time             `json:"completed_on,omitempty"`
	BuildSecondsUsed       int                    `json:"build_seconds_used"`
	ConfigurationSources   []*ConfigurationSource `json:"configuration_sources,omitempty"`
	Links                  *PipelineLinks         `json:"links,omitempty"`
}

// PipelineState represents the state of a pipeline
type PipelineState struct {
	Type   string          `json:"type"`
	Name   string          `json:"name"`
	Stage  *PipelineStage  `json:"stage,omitempty"`
	Result *PipelineResult `json:"result,omitempty"`
}

// PipelineStage represents the stage of a pipeline
type PipelineStage struct {
	Type string `json:"type"`
	Name string `json:"name"`
}

// PipelineResult represents the result of a pipeline
type PipelineResult struct {
	Type string `json:"type"`
	Name string `json:"name"`
}

// PipelineTarget represents the target of a pipeline (branch, tag, etc.)
type PipelineTarget struct {
	Type           string     `json:"type"`
	RefType        string     `json:"ref_type,omitempty"`
	RefName        string     `json:"ref_name,omitempty"`
	Selector       *Selector  `json:"selector,omitempty"`
	Commit         *Commit    `json:"commit,omitempty"`
	PullRequestId  *int       `json:"pullRequestId,omitempty"`
}

// PipelineTrigger represents what triggered the pipeline
type PipelineTrigger struct {
	Type string `json:"type"`
	Name string `json:"name"`
}

// PipelineVariable represents a pipeline variable
type PipelineVariable struct {
	Type    string `json:"type"`
	UUID    string `json:"uuid,omitempty"`
	Key     string `json:"key"`
	Value   string `json:"value"`
	Secured bool   `json:"secured,omitempty"`
}

// ConfigurationSource represents a pipeline configuration source
type ConfigurationSource struct {
	Source string `json:"source"`
	URI    string `json:"uri"`
}

// PipelineLinks represents links related to a pipeline
type PipelineLinks struct {
	Self  *Link `json:"self,omitempty"`
	Steps *Link `json:"steps,omitempty"`
}

// PipelineStep represents a step in a pipeline
type PipelineStep struct {
	Type            string                 `json:"type"`
	UUID            string                 `json:"uuid"`
	Name            string                 `json:"name,omitempty"`
	StartedOn       *time.Time             `json:"started_on,omitempty"`
	CompletedOn     *time.Time             `json:"completed_on,omitempty"`
	State           *PipelineState         `json:"state,omitempty"`
	Image           *PipelineImage         `json:"image,omitempty"`
	SetupCommands   []*PipelineCommand     `json:"setup_commands,omitempty"`
	ScriptCommands  []*PipelineCommand     `json:"script_commands,omitempty"`
	Logs            *Link                  `json:"logs,omitempty"`
	MaxTime         int                    `json:"max_time,omitempty"`
	BuildSecondsUsed int                   `json:"build_seconds_used"`
}

// PipelineImage represents the Docker image used in a pipeline step
type PipelineImage struct {
	Name     string `json:"name"`
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
	Email    string `json:"email,omitempty"`
}

// PipelineCommand represents a command in a pipeline step
type PipelineCommand struct {
	Name    string `json:"name"`
	Command string `json:"command"`
}

// User represents a Bitbucket user
type User struct {
	Type        string `json:"type"`
	UUID        string `json:"uuid,omitempty"`
	Username    string `json:"username,omitempty"`
	Nickname    string `json:"nickname,omitempty"`
	DisplayName string `json:"display_name,omitempty"`
	AccountID   string `json:"account_id,omitempty"`
	Links       *Links `json:"links,omitempty"`
}

// Repository represents a Bitbucket repository
type Repository struct {
	Type     string `json:"type"`
	UUID     string `json:"uuid,omitempty"`
	Name     string `json:"name"`
	FullName string `json:"full_name"`
	Links    *Links `json:"links,omitempty"`
}

// Selector represents a pipeline selector
type Selector struct {
	Type    string `json:"type"`
	Pattern string `json:"pattern,omitempty"`
}

// Commit represents a Git commit
type Commit struct {
	Type    string     `json:"type"`
	Hash    string     `json:"hash"`
	Message string     `json:"message,omitempty"`
	Date    *time.Time `json:"date,omitempty"`
	Author  *User      `json:"author,omitempty"`
	Links   *Links     `json:"links,omitempty"`
}

// Link represents a hypermedia link
type Link struct {
	Href string `json:"href"`
	Name string `json:"name,omitempty"`
}

// Links represents a collection of hypermedia links
type Links struct {
	Self   *Link `json:"self,omitempty"`
	HTML   *Link `json:"html,omitempty"`
	Avatar *Link `json:"avatar,omitempty"`
}

// Artifact represents a pipeline artifact
type Artifact struct {
	Type        string     `json:"type"`
	UUID        string     `json:"uuid"`
	Name        string     `json:"name"`
	Size        int64      `json:"size"`
	CreatedOn   *time.Time `json:"created_on,omitempty"`
	DownloadURL string     `json:"download_url"`
	Links       *Links     `json:"links,omitempty"`
}

// TriggerPipelineRequest represents a request to trigger a pipeline
type TriggerPipelineRequest struct {
	Target    *PipelineTarget     `json:"target"`
	Variables []*PipelineVariable `json:"variables,omitempty"`
}

// PipelineListOptions represents options for listing pipelines
type PipelineListOptions struct {
	Status   string `json:"status,omitempty"`   // PENDING, IN_PROGRESS, SUCCESSFUL, FAILED, ERROR, STOPPED
	Branch   string `json:"branch,omitempty"`   // Filter by branch name
	Sort     string `json:"sort,omitempty"`     // Sort field (created_on, -created_on)
	Page     int    `json:"page,omitempty"`     // Page number
	PageLen  int    `json:"pagelen,omitempty"`  // Items per page
}

// PipelineStateType represents the possible pipeline states
type PipelineStateType string

const (
	PipelineStatePending    PipelineStateType = "PENDING"
	PipelineStateInProgress PipelineStateType = "IN_PROGRESS"
	PipelineStateSuccessful PipelineStateType = "SUCCESSFUL"
	PipelineStateFailed     PipelineStateType = "FAILED"
	PipelineStateError      PipelineStateType = "ERROR"
	PipelineStateStopped    PipelineStateType = "STOPPED"
)

// String returns the string representation of PipelineStateType
func (p PipelineStateType) String() string {
	return string(p)
}

// TestReport represents a summary of test reports for a pipeline step
type TestReport struct {
	Type       string      `json:"type"`
	UUID       string      `json:"uuid"`
	Name       string      `json:"name,omitempty"`
	Status     string      `json:"status,omitempty"`
	Result     string      `json:"result,omitempty"`
	Passed     int         `json:"passed,omitempty"`
	Failed     int         `json:"failed,omitempty"`
	Skipped    int         `json:"skipped,omitempty"`
	Total      int         `json:"total,omitempty"`
	Duration   float64     `json:"duration,omitempty"`
	CreatedOn  *time.Time  `json:"created_on,omitempty"`
}

// TestCase represents a test case from a pipeline step
type TestCase struct {
	Type        string     `json:"type"`
	UUID        string     `json:"uuid"`
	Name        string     `json:"name"`
	ClassName   string     `json:"class_name,omitempty"`
	TestSuite   string     `json:"test_suite,omitempty"`
	Status      string     `json:"status"`
	Result      string     `json:"result,omitempty"`
	Duration    float64    `json:"duration,omitempty"`
	Message     string     `json:"message,omitempty"`
	Stacktrace  string     `json:"stacktrace,omitempty"`
}

// TestCaseReason represents the output/reason for a test case failure
type TestCaseReason struct {
	Type    string `json:"type"`
	Message string `json:"message"`
	Output  string `json:"output,omitempty"`
}