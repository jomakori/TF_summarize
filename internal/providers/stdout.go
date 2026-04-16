package providers

import (
	"fmt"

	"github.com/jomakori/TF_summarize/internal"
)

// StdoutProvider writes terraform summaries to stdout.
type StdoutProvider struct {
	*BaseProvider
}

// NewStdoutProvider creates a new stdout provider.
func NewStdoutProvider() *StdoutProvider {
	return &StdoutProvider{
		BaseProvider: NewBaseProvider("stdout"),
	}
}

// WriteSummary writes the markdown summary to stdout.
func (p *StdoutProvider) WriteSummary(summary *internal.Summary, markdown string) error {
	fmt.Print(markdown)
	return nil
}

// WriteOutputs is a no-op for stdout (outputs are included in summary).
func (p *StdoutProvider) WriteOutputs(summary *internal.Summary, markdown string) error {
	return nil
}
