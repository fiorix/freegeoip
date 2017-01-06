package httpmux

import "net/http"

// ConfigOption is the interface for updating config options.
type ConfigOption interface {
	Set(c *Config)
}

// ConfigOptionFunc is an adapter for config option functions.
type ConfigOptionFunc func(c *Config)

// Set implements the ConfigOption interface.
func (f ConfigOptionFunc) Set(c *Config) { f(c) }

// WithPrefix returns a ConfigOption that uptates the Config.
func WithPrefix(prefix string) ConfigOption {
	return ConfigOptionFunc(func(c *Config) { c.Prefix = prefix })
}

// WithMiddleware returns a ConfigOption that uptates the Config.
func WithMiddleware(mw ...Middleware) ConfigOption {
	return ConfigOptionFunc(func(c *Config) { c.Middleware = mw })
}

// WithMiddlewareFunc returns a ConfigOption that uptates the Config.
func WithMiddlewareFunc(mw ...MiddlewareFunc) ConfigOption {
	return ConfigOptionFunc(func(c *Config) { c.UseFunc(mw...) })
}

// WithRedirectTrailingSlash returns a ConfigOption that uptates the Config.
func WithRedirectTrailingSlash(v bool) ConfigOption {
	return ConfigOptionFunc(func(c *Config) { c.RedirectTrailingSlash = v })
}

// WithRedirectFixedPath returns a ConfigOption that uptates the Config.
func WithRedirectFixedPath(v bool) ConfigOption {
	return ConfigOptionFunc(func(c *Config) { c.RedirectFixedPath = v })
}

// WithHandleMethodNotAllowed returns a ConfigOption that uptates the Config.
func WithHandleMethodNotAllowed(v bool) ConfigOption {
	return ConfigOptionFunc(func(c *Config) { c.HandleMethodNotAllowed = v })
}

// WithNotFound returns a ConfigOption that uptates the Config.
func WithNotFound(f http.Handler) ConfigOption {
	return ConfigOptionFunc(func(c *Config) { c.NotFound = f })
}

// WithMethodNotAllowed returns a ConfigOption that uptates the Config.
func WithMethodNotAllowed(f http.Handler) ConfigOption {
	return ConfigOptionFunc(func(c *Config) { c.MethodNotAllowed = f })
}

// WithPanicHandler returns a ConfigOption that uptates the Config.
func WithPanicHandler(f func(http.ResponseWriter, *http.Request, interface{})) ConfigOption {
	return ConfigOptionFunc(func(c *Config) { c.PanicHandler = f })
}
