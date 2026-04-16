package providers

import (
	"github.com/jomakori/TF_summarize/internal"
)

// BaseProvider provides common functionality for all output providers.
type BaseProvider struct {
	name string
}

// NewBaseProvider creates a new base provider.
func NewBaseProvider(name string) *BaseProvider {
	return &BaseProvider{name: name}
}

// Name returns the provider name.
func (p *BaseProvider) Name() string {
	return p.name
}

// WriteSummary is a no-op in the base provider.
func (p *BaseProvider) WriteSummary(summary *internal.Summary, markdown string) error {
	return nil
}

// WriteOutputs is a no-op in the base provider.
func (p *BaseProvider) WriteOutputs(summary *internal.Summary, markdown string) error {
	return nil
}
