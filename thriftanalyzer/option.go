package thriftanalyzer

// analysisOptions holds the internal configuration for the analyzer.
// It's not exported to keep it private to the package.
type analysisOptions struct {
	scopes []string
}

// Option is the functional option type.
type Option func(*analysisOptions)

// newDefaultOptions creates the default internal configuration.
func newDefaultOptions() *analysisOptions {
	return &analysisOptions{
		scopes: []string{"go"}, // Default is still to collect only 'go' namespaces.
	}
}

// WithScopes is a functional option to specify which namespace scopes to collect.
// If not used, the default is `[]string{"go"}`.
// To collect all namespaces, provide a slice containing just "*".
// Example: thriftanalyzer.WithScopes("go", "java")
func WithScopes(scopes ...string) Option {
	return func(opts *analysisOptions) {
		if len(scopes) == 0 {
			// If called with no arguments, revert to default.
			opts.scopes = []string{"go"}
		} else {
			opts.scopes = scopes
		}
	}
}

// WithAllScopes is a convenience option to collect namespaces for all language scopes.
// It's equivalent to WithScopes("*").
func WithAllScopes() Option {
	return func(opts *analysisOptions) {
		opts.scopes = []string{"*"}
	}
}
