package swagger2thrift

// Config holds the configuration for the conversion process.
type Config struct {
	// Namespace for the generated Go code. If empty, it will be auto-generated
	// from the input filename.
	Namespace string
	// ServiceName specifies the name for the main aggregated service.
	// If left empty, it defaults to "HTTPService".
	ServiceName string
}

// Option is a function that applies a configuration option to a Config object.
type Option func(*Config)

// WithNamespace sets a custom namespace for the generated Go code.
func WithNamespace(namespace string) Option {
	return func(c *Config) {
		c.Namespace = namespace
	}
}

// WithServiceName sets a custom name for the generated service.
func WithServiceName(serviceName string) Option {
	return func(c *Config) {
		c.ServiceName = serviceName
	}
}
