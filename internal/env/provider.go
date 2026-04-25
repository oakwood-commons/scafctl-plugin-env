// Package env implements an environment variable provider plugin for scafctl.
package env

import (
	"context"
	"fmt"
	"os"

	"github.com/Masterminds/semver/v3"
	"github.com/go-logr/logr"
	"github.com/google/jsonschema-go/jsonschema"

	sdkplugin "github.com/oakwood-commons/scafctl-plugin-sdk/plugin"
	sdkprovider "github.com/oakwood-commons/scafctl-plugin-sdk/provider"
	"github.com/oakwood-commons/scafctl-plugin-sdk/provider/schemahelper"
)

const (
	// ProviderName is the name of the environment provider.
	ProviderName = "env"
	// Version is the version of the environment provider.
	Version = "1.0.0"
)

// Ops defines the interface for environment variable operations.
type Ops interface {
	LookupEnv(key string) (string, bool)
	Setenv(key, value string) error
	Unsetenv(key string) error
	Environ() []string
}

// DefaultEnvOps provides real OS environment operations.
type DefaultEnvOps struct{}

// LookupEnv looks up an environment variable.
func (d *DefaultEnvOps) LookupEnv(key string) (string, bool) {
	return os.LookupEnv(key)
}

// Setenv sets an environment variable.
func (d *DefaultEnvOps) Setenv(key, value string) error {
	return os.Setenv(key, value)
}

// Unsetenv unsets an environment variable.
func (d *DefaultEnvOps) Unsetenv(key string) error {
	return os.Unsetenv(key)
}

// Environ returns all environment variables.
func (d *DefaultEnvOps) Environ() []string {
	return os.Environ()
}

// Plugin implements the ProviderPlugin interface for environment variable operations.
type Plugin struct {
	envOps Ops
}

// Option is a functional option for configuring Plugin.
type Option func(*Plugin)

// WithOps sets custom environment operations (for testing).
func WithOps(ops Ops) Option {
	return func(p *Plugin) {
		p.envOps = ops
	}
}

// NewPlugin creates a new environment variable plugin with the given options.
func NewPlugin(opts ...Option) *Plugin {
	p := &Plugin{}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

// ops returns the Ops implementation, defaulting to real OS operations.
func (p *Plugin) ops() Ops {
	if p.envOps != nil {
		return p.envOps
	}
	return &DefaultEnvOps{}
}

var version = func() *semver.Version {
	v, _ := semver.NewVersion(Version)
	return v
}()

func intPtr(v int) *int { return &v }

func buildDescriptor() *sdkprovider.Descriptor {
	return &sdkprovider.Descriptor{
		Name:        ProviderName,
		DisplayName: "Environment Variables",
		APIVersion:  "v1",
		Description: "Provider for reading and setting environment variables",
		Version:     version,
		Category:    "system",
		Capabilities: []sdkprovider.Capability{
			sdkprovider.CapabilityFrom,
		},
		Schema: schemahelper.ObjectSchema([]string{"operation"}, map[string]*jsonschema.Schema{
			"operation": schemahelper.StringProp("Operation to perform: 'get' to read a variable, 'set' to set a variable, 'list' to list all variables, 'unset' to remove a variable",
				schemahelper.WithEnum("get", "set", "list", "unset"),
				schemahelper.WithExample("get"),
				schemahelper.WithMaxLength(*intPtr(10))),
			"name": schemahelper.StringProp("Name of the environment variable (required for get, set, unset operations)",
				schemahelper.WithMaxLength(*intPtr(256)),
				schemahelper.WithPattern(`^[A-Za-z_][A-Za-z0-9_]*$`),
				schemahelper.WithExample("HOME")),
			"value": schemahelper.StringProp("Value to set (required for set operation)",
				schemahelper.WithMaxLength(*intPtr(4096)),
				schemahelper.WithExample("/home/user")),
			"default": schemahelper.StringProp("Default value to return if variable is not set (only for get operation)",
				schemahelper.WithMaxLength(*intPtr(4096)),
				schemahelper.WithExample("default-value")),
			"prefix": schemahelper.StringProp("Filter environment variables by prefix (only for list operation)",
				schemahelper.WithMaxLength(*intPtr(256)),
				schemahelper.WithExample("AWS_")),
			"raw": schemahelper.BoolProp("Return just the value string instead of the full result map. Only applies in resolver/transform mode"),
		}),
		OutputSchemas: map[sdkprovider.Capability]*jsonschema.Schema{
			sdkprovider.CapabilityFrom: schemahelper.AnyProp("Full result map (operation, name, value, exists) by default; value string when raw: true"),
		},
		Examples: []sdkprovider.Example{
			{
				Name:        "Get environment variable",
				Description: "Read an environment variable with a default value fallback",
				YAML: `name: get-home
provider: env
inputs:
  operation: get
  name: HOME
  default: "/home/default"`,
			},
			{
				Name:        "Set environment variable",
				Description: "Set an environment variable for the current process",
				YAML: `name: set-api-key
provider: env
inputs:
  operation: set
  name: API_KEY
  value: "secret-key-value"`,
			},
			{
				Name:        "List environment variables",
				Description: "List all environment variables with a specific prefix",
				YAML: `name: list-aws-vars
provider: env
inputs:
  operation: list
  prefix: "AWS_"`,
			},
			{
				Name:        "Unset environment variable",
				Description: "Remove an environment variable from the current process",
				YAML: `name: unset-temp-var
provider: env
inputs:
  operation: unset
  name: TEMP_VAR`,
			},
		},
	}
}

// GetProviders returns the list of provider names this plugin offers.
func (p *Plugin) GetProviders(_ context.Context) ([]string, error) {
	return []string{ProviderName}, nil
}

// GetProviderDescriptor returns the descriptor for the named provider.
func (p *Plugin) GetProviderDescriptor(_ context.Context, providerName string) (*sdkprovider.Descriptor, error) {
	if providerName != ProviderName {
		return nil, fmt.Errorf("unknown provider: %s", providerName)
	}
	return buildDescriptor(), nil
}

// ConfigureProvider configures the provider (no-op for env).
func (p *Plugin) ConfigureProvider(_ context.Context, _ string, _ sdkplugin.ProviderConfig) error {
	return nil
}

// ExecuteProvider performs the environment variable operation.
func (p *Plugin) ExecuteProvider(ctx context.Context, providerName string, inputs map[string]any) (*sdkprovider.Output, error) {
	if providerName != ProviderName {
		return nil, fmt.Errorf("unknown provider: %s", providerName)
	}

	lgr := logr.FromContextOrDiscard(ctx)

	operation, ok := inputs["operation"].(string)
	if !ok {
		return nil, fmt.Errorf("%s: operation is required and must be a string", ProviderName)
	}

	lgr.V(1).Info("executing provider", "provider", ProviderName, "operation", operation)

	if dryRun := sdkprovider.DryRunFromContext(ctx); dryRun {
		return p.executeDryRun(operation, inputs)
	}

	var result *sdkprovider.Output
	var err error

	switch operation {
	case "get":
		result, err = p.executeGet(ctx, inputs)
	case "set":
		result, err = p.executeSet(inputs)
	case "list":
		result, err = p.executeList(inputs)
	case "unset":
		result, err = p.executeUnset(inputs)
	default:
		return nil, fmt.Errorf("%s: unsupported operation: %s", ProviderName, operation)
	}

	if err != nil {
		return nil, fmt.Errorf("%s: %w", ProviderName, err)
	}

	lgr.V(1).Info("provider execution completed", "provider", ProviderName, "operation", operation)

	return result, nil
}

// ExecuteProviderStream is not supported by the env provider.
func (p *Plugin) ExecuteProviderStream(_ context.Context, _ string, _ map[string]any, _ func(sdkplugin.StreamChunk)) error {
	return sdkplugin.ErrStreamingNotSupported
}

// DescribeWhatIf returns a human-readable description of what the operation would do.
func (p *Plugin) DescribeWhatIf(_ context.Context, providerName string, inputs map[string]any) (string, error) {
	if providerName != ProviderName {
		return "", fmt.Errorf("unknown provider: %s", providerName)
	}

	operation, _ := inputs["operation"].(string)
	name, _ := inputs["name"].(string)

	switch operation {
	case "get":
		dflt, _ := inputs["default"].(string)
		if dflt != "" {
			return fmt.Sprintf("Read environment variable %q (default: %q)", name, dflt), nil
		}
		return fmt.Sprintf("Read environment variable %q", name), nil
	case "set":
		return fmt.Sprintf("Set environment variable %q (current process only)", name), nil
	case "list":
		prefix, _ := inputs["prefix"].(string)
		return fmt.Sprintf("List environment variables with prefix %q", prefix), nil
	case "unset":
		return fmt.Sprintf("Remove environment variable %q (current process only)", name), nil
	default:
		return fmt.Sprintf("Unknown env operation: %s", operation), nil
	}
}

// ExtractDependencies returns nil (env has no external dependencies).
func (p *Plugin) ExtractDependencies(_ context.Context, _ string, _ map[string]any) ([]string, error) {
	return nil, nil
}

// StopProvider is a no-op for the env provider.
func (p *Plugin) StopProvider(_ context.Context, _ string) error {
	return nil
}

func (p *Plugin) executeGet(ctx context.Context, inputs map[string]any) (*sdkprovider.Output, error) {
	name, ok := inputs["name"].(string)
	if !ok || name == "" {
		return nil, fmt.Errorf("name is required for get operation")
	}

	value, exists := p.ops().LookupEnv(name)
	if !exists {
		if defaultValue, ok := inputs["default"].(string); ok {
			value = defaultValue
		}
	}

	raw, _ := inputs["raw"].(bool)
	if raw {
		if mode, modeOK := sdkprovider.ExecutionModeFromContext(ctx); modeOK &&
			(mode == sdkprovider.CapabilityFrom || mode == sdkprovider.CapabilityTransform) {
			return &sdkprovider.Output{Data: value}, nil
		}
	}

	return &sdkprovider.Output{
		Data: map[string]any{
			"operation": "get",
			"name":      name,
			"value":     value,
			"exists":    exists,
		},
	}, nil
}

func (p *Plugin) executeSet(inputs map[string]any) (*sdkprovider.Output, error) {
	name, ok := inputs["name"].(string)
	if !ok || name == "" {
		return nil, fmt.Errorf("name is required for set operation")
	}

	value, ok := inputs["value"].(string)
	if !ok {
		return nil, fmt.Errorf("value is required for set operation")
	}

	if err := p.ops().Setenv(name, value); err != nil {
		return nil, fmt.Errorf("failed to set environment variable: %w", err)
	}

	return &sdkprovider.Output{
		Data: map[string]any{
			"operation": "set",
			"name":      name,
			"value":     value,
		},
		Metadata: map[string]any{
			"warning": "Environment variable changes only affect the current process",
		},
	}, nil
}

//nolint:unparam // Error return kept for consistent interface
func (p *Plugin) executeList(inputs map[string]any) (*sdkprovider.Output, error) {
	prefix, _ := inputs["prefix"].(string)
	if prefix == "" {
		return nil, fmt.Errorf("prefix is required for list operation: listing all environment variables without a scope would expose process secrets")
	}

	envVars := make(map[string]string)

	for _, env := range p.ops().Environ() {
		for i := range len(env) {
			if env[i] == '=' {
				key := env[:i]
				value := env[i+1:]

				if len(key) >= len(prefix) && key[:len(prefix)] == prefix {
					envVars[key] = value
				}
				break
			}
		}
	}

	return &sdkprovider.Output{
		Data: map[string]any{
			"operation": "list",
			"variables": envVars,
			"count":     len(envVars),
		},
	}, nil
}

func (p *Plugin) executeUnset(inputs map[string]any) (*sdkprovider.Output, error) {
	name, ok := inputs["name"].(string)
	if !ok || name == "" {
		return nil, fmt.Errorf("name is required for unset operation")
	}

	if err := p.ops().Unsetenv(name); err != nil {
		return nil, fmt.Errorf("failed to unset environment variable: %w", err)
	}

	return &sdkprovider.Output{
		Data: map[string]any{
			"operation": "unset",
			"name":      name,
		},
		Metadata: map[string]any{
			"warning": "Environment variable changes only affect the current process",
		},
	}, nil
}

func (p *Plugin) executeDryRun(operation string, inputs map[string]any) (*sdkprovider.Output, error) {
	result := map[string]any{
		"operation": operation,
	}

	switch operation {
	case "get":
		if name, ok := inputs["name"].(string); ok {
			result["name"] = name
			result["value"] = "[DRY-RUN] Value not retrieved"
			result["exists"] = false
		}
	case "set":
		if name, ok := inputs["name"].(string); ok {
			result["name"] = name
		}
		if value, ok := inputs["value"].(string); ok {
			result["value"] = value
		}
	case "list":
		result["variables"] = map[string]string{}
		result["count"] = 0
	case "unset":
		if name, ok := inputs["name"].(string); ok {
			result["name"] = name
		}
	}

	return &sdkprovider.Output{
		Data: result,
		Metadata: map[string]any{
			"dryRun": true,
		},
	}, nil
}
