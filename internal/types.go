package internal

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
	Address string
	Action  Action
	// For apply phase: did this resource succeed or fail?
	Success bool
	Error   string
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
}

// OutputTarget is where the summary gets written.
type OutputTarget string

const (
	TargetGHASummary OutputTarget = "gha"
	TargetPR         OutputTarget = "pr"
	TargetStdout     OutputTarget = "stdout"
)
