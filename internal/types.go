package internal

import (
	"time"
)

// Phase represents whether this is a plan or apply run.
type Phase string

const (
	PhasePlan  Phase = "plan"
	PhaseApply Phase = "apply"
)

// Action categorizes what happens to a resource.
type Action string

const (
	ActionCreate  Action = "create"
	ActionUpdate  Action = "update"
	ActionDestroy Action = "destroy"
	ActionReplace Action = "replace" // destroy + create
	ActionRead    Action = "read"
	ActionImport  Action = "import"
)

// ResourceChange is a single resource affected by the plan/apply.
type ResourceChange struct {
	Address   string
	Action    Action
	Success   bool
	Error     string
	Timestamp time.Time
	Details   map[string]interface{} // For future extensibility
}

// Summary holds the parsed result of a terraform plan or apply output.
type Summary struct {
	Phase     Phase
	Workspace string

	// Whether this is a destroy plan (terraform plan -destroy)
	IsDestroyPlan bool

	// Counts
	ToAdd     int
	ToChange  int
	ToDestroy int
	ToImport  int

	// Resources grouped by action
	Creates  []ResourceChange
	Updates  []ResourceChange
	Destroys []ResourceChange
	Replaces []ResourceChange
	Reads    []ResourceChange
	Imports  []ResourceChange

	// Apply-specific: resources that failed during apply
	Failures []ResourceChange

	// Apply-specific
	ApplySucceeded bool
	ApplyError     string
	Applied        int
	Failed         int

	// Warnings/errors from the output
	Warnings []string
	Errors   []string

	// Raw output for the detail dropdown (full terraform output)
	RawOutput string

	// Drift detected
	DriftDetected bool

	// Execution error (if command failed)
	ExecutionError error
	ErrorContext   string

	// Parsed from JSON plan (if available)
	ParsedFromJSON bool
}

// OutputTarget is where the summary gets written.
type OutputTarget string

const (
	TargetGHASummary OutputTarget = "gha"
	TargetPR         OutputTarget = "pr"
	TargetStdout     OutputTarget = "stdout"
)

// OutputProvider defines the interface for writing terraform summaries to different targets.
type OutputProvider interface {
	// WriteSummary writes the markdown summary to the target.
	WriteSummary(summary *Summary, markdown string) error

	// WriteOutputs writes terraform outputs (if any) to the target.
	WriteOutputs(summary *Summary, markdown string) error

	// Name returns the provider name for logging.
	Name() string
}

// RenderOutput holds the different rendered sections.
type RenderOutput struct {
	Summary    string // Summary with badges and counts
	Details    string // Resource lists
	Outputs    string // Terraform outputs
	RawOutput  string // Full terraform output
	Full       string // Complete markdown
}
