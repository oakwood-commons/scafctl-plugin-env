package env

import (
	"context"
	"testing"

	sdkplugin "github.com/oakwood-commons/scafctl-plugin-sdk/plugin"
	sdkprovider "github.com/oakwood-commons/scafctl-plugin-sdk/provider"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetProviders(t *testing.T) {
	p := NewPlugin()
	names, err := p.GetProviders(context.Background())
	require.NoError(t, err)
	assert.Equal(t, []string{"env"}, names)
}

func TestGetProviderDescriptor(t *testing.T) {
	p := NewPlugin()
	desc, err := p.GetProviderDescriptor(context.Background(), "env")
	require.NoError(t, err)
	require.NotNil(t, desc)

	assert.Equal(t, "env", desc.Name)
	assert.Equal(t, "1.0.0", desc.Version.String())
	assert.Equal(t, "system", desc.Category)
	assert.Contains(t, desc.Capabilities, sdkprovider.CapabilityFrom)
	assert.NotNil(t, desc.Schema)
	assert.NotEmpty(t, desc.Schema.Properties)
	assert.NotNil(t, desc.OutputSchemas)
	assert.NotNil(t, desc.OutputSchemas[sdkprovider.CapabilityFrom])
}

func TestGetProviderDescriptor_Unknown(t *testing.T) {
	p := NewPlugin()
	_, err := p.GetProviderDescriptor(context.Background(), "nope")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown provider")
}

func TestExecuteProvider_Get(t *testing.T) {
	mockOps := NewMockEnvOps()
	p := NewPlugin(WithEnvOps(mockOps))

	testKey := "TEST_ENV_VAR_GET"
	testValue := "test-value-123"
	mockOps.Set(testKey, testValue)

	output, err := p.ExecuteProvider(context.Background(), "env", map[string]any{
		"operation": "get",
		"name":      testKey,
	})
	require.NoError(t, err)
	require.NotNil(t, output)

	data := output.Data.(map[string]any)
	assert.Equal(t, "get", data["operation"])
	assert.Equal(t, testKey, data["name"])
	assert.Equal(t, testValue, data["value"])
	assert.Equal(t, true, data["exists"])
}

func TestExecuteProvider_Get_NotExists(t *testing.T) {
	mockOps := NewMockEnvOps()
	p := NewPlugin(WithEnvOps(mockOps))

	output, err := p.ExecuteProvider(context.Background(), "env", map[string]any{
		"operation": "get",
		"name":      "TEST_ENV_VAR_NOT_EXISTS",
	})
	require.NoError(t, err)
	require.NotNil(t, output)

	data := output.Data.(map[string]any)
	assert.Equal(t, "get", data["operation"])
	assert.Equal(t, "", data["value"])
	assert.Equal(t, false, data["exists"])
}

func TestExecuteProvider_Get_WithDefault(t *testing.T) {
	mockOps := NewMockEnvOps()
	p := NewPlugin(WithEnvOps(mockOps))

	output, err := p.ExecuteProvider(context.Background(), "env", map[string]any{
		"operation": "get",
		"name":      "TEST_ENV_VAR_DEFAULT",
		"default":   "default-value",
	})
	require.NoError(t, err)
	require.NotNil(t, output)

	data := output.Data.(map[string]any)
	assert.Equal(t, "default-value", data["value"])
	assert.Equal(t, false, data["exists"])
}

func TestExecuteProvider_Get_Raw(t *testing.T) {
	mockOps := NewMockEnvOps()
	p := NewPlugin(WithEnvOps(mockOps))

	mockOps.Set("RAW_VAR", "raw-value")

	ctx := sdkprovider.WithExecutionMode(context.Background(), sdkprovider.CapabilityFrom)

	output, err := p.ExecuteProvider(ctx, "env", map[string]any{
		"operation": "get",
		"name":      "RAW_VAR",
		"raw":       true,
	})
	require.NoError(t, err)
	require.NotNil(t, output)
	assert.Equal(t, "raw-value", output.Data)
}

func TestExecuteProvider_Get_Raw_TransformMode(t *testing.T) {
	mockOps := NewMockEnvOps()
	p := NewPlugin(WithEnvOps(mockOps))

	mockOps.Set("RAW_VAR2", "raw-transform")

	ctx := sdkprovider.WithExecutionMode(context.Background(), sdkprovider.CapabilityTransform)

	output, err := p.ExecuteProvider(ctx, "env", map[string]any{
		"operation": "get",
		"name":      "RAW_VAR2",
		"raw":       true,
	})
	require.NoError(t, err)
	require.NotNil(t, output)
	assert.Equal(t, "raw-transform", output.Data)
}

func TestExecuteProvider_Get_Raw_NoMode(t *testing.T) {
	mockOps := NewMockEnvOps()
	p := NewPlugin(WithEnvOps(mockOps))

	mockOps.Set("RAW_VAR3", "should-be-map")

	// No execution mode in context -- raw should be ignored
	output, err := p.ExecuteProvider(context.Background(), "env", map[string]any{
		"operation": "get",
		"name":      "RAW_VAR3",
		"raw":       true,
	})
	require.NoError(t, err)
	require.NotNil(t, output)
	data := output.Data.(map[string]any)
	assert.Equal(t, "should-be-map", data["value"])
}

func TestExecuteProvider_Set(t *testing.T) {
	mockOps := NewMockEnvOps()
	p := NewPlugin(WithEnvOps(mockOps))

	testKey := "TEST_ENV_VAR_SET"
	testValue := "new-value-456"

	output, err := p.ExecuteProvider(context.Background(), "env", map[string]any{
		"operation": "set",
		"name":      testKey,
		"value":     testValue,
	})
	require.NoError(t, err)
	require.NotNil(t, output)

	actualValue, exists := mockOps.Get(testKey)
	assert.True(t, exists)
	assert.Equal(t, testValue, actualValue)

	data := output.Data.(map[string]any)
	assert.Equal(t, "set", data["operation"])
	assert.Equal(t, testKey, data["name"])
	assert.Equal(t, testValue, data["value"])
	assert.NotNil(t, output.Metadata)
	assert.Contains(t, output.Metadata, "warning")
}

func TestExecuteProvider_List_WithoutPrefix(t *testing.T) {
	mockOps := NewMockEnvOps()
	p := NewPlugin(WithEnvOps(mockOps))

	mockOps.Set("PATH", "/usr/bin")

	output, err := p.ExecuteProvider(context.Background(), "env", map[string]any{
		"operation": "list",
	})
	require.Error(t, err)
	assert.Nil(t, output)
	assert.Contains(t, err.Error(), "prefix is required")
}

func TestExecuteProvider_List_WithPrefix(t *testing.T) {
	mockOps := NewMockEnvOps()
	p := NewPlugin(WithEnvOps(mockOps))

	mockOps.Set("TEST_PREFIX_VAR1", "value1")
	mockOps.Set("TEST_PREFIX_VAR2", "value2")
	mockOps.Set("TEST_PREFIX_VAR3", "value3")
	mockOps.Set("OTHER_VAR", "should-not-appear")

	output, err := p.ExecuteProvider(context.Background(), "env", map[string]any{
		"operation": "list",
		"prefix":    "TEST_PREFIX_",
	})
	require.NoError(t, err)
	require.NotNil(t, output)

	data := output.Data.(map[string]any)
	variables := data["variables"].(map[string]string)
	count := data["count"].(int)

	assert.Equal(t, 3, count)
	assert.Equal(t, "value1", variables["TEST_PREFIX_VAR1"])
	assert.Equal(t, "value2", variables["TEST_PREFIX_VAR2"])
	assert.Equal(t, "value3", variables["TEST_PREFIX_VAR3"])
	assert.NotContains(t, variables, "OTHER_VAR")
}

func TestExecuteProvider_Unset(t *testing.T) {
	mockOps := NewMockEnvOps()
	p := NewPlugin(WithEnvOps(mockOps))

	testKey := "TEST_ENV_VAR_UNSET"
	mockOps.Set(testKey, "to-be-removed")

	_, exists := mockOps.Get(testKey)
	assert.True(t, exists)

	output, err := p.ExecuteProvider(context.Background(), "env", map[string]any{
		"operation": "unset",
		"name":      testKey,
	})
	require.NoError(t, err)
	require.NotNil(t, output)

	_, exists = mockOps.Get(testKey)
	assert.False(t, exists)

	data := output.Data.(map[string]any)
	assert.Equal(t, "unset", data["operation"])
	assert.Equal(t, testKey, data["name"])
	assert.NotNil(t, output.Metadata)
	assert.Contains(t, output.Metadata, "warning")
}

func TestExecuteProvider_DryRun_Get(t *testing.T) {
	mockOps := NewMockEnvOps()
	p := NewPlugin(WithEnvOps(mockOps))
	ctx := sdkprovider.WithDryRun(context.Background(), true)

	output, err := p.ExecuteProvider(ctx, "env", map[string]any{
		"operation": "get",
		"name":      "TEST_VAR",
	})
	require.NoError(t, err)
	require.NotNil(t, output)

	data := output.Data.(map[string]any)
	assert.Equal(t, "get", data["operation"])
	assert.Equal(t, "TEST_VAR", data["name"])
	assert.Contains(t, data["value"], "DRY-RUN")
	assert.Equal(t, false, data["exists"])
	assert.Equal(t, true, output.Metadata["dryRun"])
}

func TestExecuteProvider_DryRun_Set(t *testing.T) {
	mockOps := NewMockEnvOps()
	p := NewPlugin(WithEnvOps(mockOps))
	ctx := sdkprovider.WithDryRun(context.Background(), true)

	testKey := "TEST_VAR_SET_DRYRUN"

	output, err := p.ExecuteProvider(ctx, "env", map[string]any{
		"operation": "set",
		"name":      testKey,
		"value":     "should-not-be-set",
	})
	require.NoError(t, err)
	require.NotNil(t, output)

	_, exists := mockOps.Get(testKey)
	assert.False(t, exists, "Variable should not be set in dry-run mode")

	data := output.Data.(map[string]any)
	assert.Equal(t, "set", data["operation"])
	assert.Equal(t, testKey, data["name"])
	assert.Equal(t, "should-not-be-set", data["value"])
	assert.Equal(t, true, output.Metadata["dryRun"])
}

func TestExecuteProvider_DryRun_List(t *testing.T) {
	mockOps := NewMockEnvOps()
	p := NewPlugin(WithEnvOps(mockOps))
	ctx := sdkprovider.WithDryRun(context.Background(), true)

	output, err := p.ExecuteProvider(ctx, "env", map[string]any{
		"operation": "list",
	})
	require.NoError(t, err)
	require.NotNil(t, output)

	data := output.Data.(map[string]any)
	assert.Equal(t, "list", data["operation"])

	variables := data["variables"].(map[string]string)
	count := data["count"].(int)
	assert.Empty(t, variables)
	assert.Equal(t, 0, count)
	assert.Equal(t, true, output.Metadata["dryRun"])
}

func TestExecuteProvider_DryRun_Unset(t *testing.T) {
	mockOps := NewMockEnvOps()
	p := NewPlugin(WithEnvOps(mockOps))
	ctx := sdkprovider.WithDryRun(context.Background(), true)

	testKey := "TEST_VAR_UNSET_DRYRUN"
	mockOps.Set(testKey, "should-remain")

	output, err := p.ExecuteProvider(ctx, "env", map[string]any{
		"operation": "unset",
		"name":      testKey,
	})
	require.NoError(t, err)
	require.NotNil(t, output)

	value, exists := mockOps.Get(testKey)
	assert.True(t, exists, "Variable should still exist in dry-run mode")
	assert.Equal(t, "should-remain", value)

	data := output.Data.(map[string]any)
	assert.Equal(t, "unset", data["operation"])
	assert.Equal(t, testKey, data["name"])
	assert.Equal(t, true, output.Metadata["dryRun"])
}

func TestExecuteProvider_InvalidOperation(t *testing.T) {
	p := NewPlugin(WithEnvOps(NewMockEnvOps()))

	output, err := p.ExecuteProvider(context.Background(), "env", map[string]any{
		"operation": "invalid",
	})
	require.Error(t, err)
	assert.Nil(t, output)
	assert.Contains(t, err.Error(), "unsupported operation")
}

func TestExecuteProvider_MissingOperationField(t *testing.T) {
	p := NewPlugin(WithEnvOps(NewMockEnvOps()))

	output, err := p.ExecuteProvider(context.Background(), "env", map[string]any{
		"name": "TEST_VAR",
	})
	require.Error(t, err)
	assert.Nil(t, output)
	assert.Contains(t, err.Error(), "operation is required")
}

func TestExecuteProvider_Get_MissingName(t *testing.T) {
	p := NewPlugin(WithEnvOps(NewMockEnvOps()))

	output, err := p.ExecuteProvider(context.Background(), "env", map[string]any{
		"operation": "get",
	})
	require.Error(t, err)
	assert.Nil(t, output)
	assert.Contains(t, err.Error(), "name is required")
}

func TestExecuteProvider_Set_MissingValue(t *testing.T) {
	p := NewPlugin(WithEnvOps(NewMockEnvOps()))

	output, err := p.ExecuteProvider(context.Background(), "env", map[string]any{
		"operation": "set",
		"name":      "TEST_VAR",
	})
	require.Error(t, err)
	assert.Nil(t, output)
	assert.Contains(t, err.Error(), "value is required")
}

func TestExecuteProvider_Unset_MissingName(t *testing.T) {
	p := NewPlugin(WithEnvOps(NewMockEnvOps()))

	output, err := p.ExecuteProvider(context.Background(), "env", map[string]any{
		"operation": "unset",
	})
	require.Error(t, err)
	assert.Nil(t, output)
	assert.Contains(t, err.Error(), "name is required")
}

func TestExecuteProvider_Set_Error(t *testing.T) {
	mockOps := NewMockEnvOps()
	mockOps.SetenvErr = true
	p := NewPlugin(WithEnvOps(mockOps))

	output, err := p.ExecuteProvider(context.Background(), "env", map[string]any{
		"operation": "set",
		"name":      "TEST_VAR",
		"value":     "test",
	})
	require.Error(t, err)
	assert.Nil(t, output)
	assert.Contains(t, err.Error(), "failed to set environment variable")
}

func TestExecuteProvider_Unset_Error(t *testing.T) {
	mockOps := NewMockEnvOps()
	mockOps.UnsetenvErr = true
	p := NewPlugin(WithEnvOps(mockOps))

	output, err := p.ExecuteProvider(context.Background(), "env", map[string]any{
		"operation": "unset",
		"name":      "TEST_VAR",
	})
	require.Error(t, err)
	assert.Nil(t, output)
	assert.Contains(t, err.Error(), "failed to unset environment variable")
}

func TestExecuteProvider_UnknownProvider(t *testing.T) {
	p := NewPlugin()

	output, err := p.ExecuteProvider(context.Background(), "nope", map[string]any{
		"operation": "get",
		"name":      "X",
	})
	require.Error(t, err)
	assert.Nil(t, output)
	assert.Contains(t, err.Error(), "unknown provider")
}

func TestDefaultEnvOps_LookupEnv(t *testing.T) {
	d := &DefaultEnvOps{}
	t.Setenv("TEST_LOOKUP_KEY", "test-value")
	val, ok := d.LookupEnv("TEST_LOOKUP_KEY")
	assert.True(t, ok)
	assert.Equal(t, "test-value", val)

	_, ok = d.LookupEnv("NON_EXISTENT_KEY_XYZ123")
	assert.False(t, ok)
}

func TestDefaultEnvOps_Setenv(t *testing.T) {
	d := &DefaultEnvOps{}
	err := d.Setenv("TEST_SET_KEY", "set-value")
	assert.NoError(t, err)
}

func TestDefaultEnvOps_Unsetenv(t *testing.T) {
	d := &DefaultEnvOps{}
	t.Setenv("TEST_UNSET_KEY", "unset-value")
	err := d.Unsetenv("TEST_UNSET_KEY")
	assert.NoError(t, err)
}

func TestDescribeWhatIf(t *testing.T) {
	p := NewPlugin()

	tests := []struct {
		name     string
		inputs   map[string]any
		contains string
	}{
		{
			name:     "get",
			inputs:   map[string]any{"operation": "get", "name": "HOME"},
			contains: `Read environment variable "HOME"`,
		},
		{
			name:     "get with default",
			inputs:   map[string]any{"operation": "get", "name": "HOME", "default": "/tmp"},
			contains: `default: "/tmp"`,
		},
		{
			name:     "set",
			inputs:   map[string]any{"operation": "set", "name": "FOO"},
			contains: `Set environment variable "FOO"`,
		},
		{
			name:     "list",
			inputs:   map[string]any{"operation": "list", "prefix": "AWS_"},
			contains: `prefix "AWS_"`,
		},
		{
			name:     "unset",
			inputs:   map[string]any{"operation": "unset", "name": "BAR"},
			contains: `Remove environment variable "BAR"`,
		},
		{
			name:     "unknown op",
			inputs:   map[string]any{"operation": "bogus"},
			contains: "Unknown env operation",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			desc, err := p.DescribeWhatIf(context.Background(), "env", tc.inputs)
			require.NoError(t, err)
			assert.Contains(t, desc, tc.contains)
		})
	}
}

func TestDescribeWhatIf_UnknownProvider(t *testing.T) {
	p := NewPlugin()
	_, err := p.DescribeWhatIf(context.Background(), "nope", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown provider")
}

func TestConfigureProvider(t *testing.T) {
	p := NewPlugin()
	err := p.ConfigureProvider(context.Background(), "env", sdkplugin.ProviderConfig{})
	assert.NoError(t, err)
}

func TestExtractDependencies(t *testing.T) {
	p := NewPlugin()
	deps, err := p.ExtractDependencies(context.Background(), "env", nil)
	assert.NoError(t, err)
	assert.Nil(t, deps)
}

func TestStopProvider(t *testing.T) {
	p := NewPlugin()
	err := p.StopProvider(context.Background(), "env")
	assert.NoError(t, err)
}

func TestExecuteProviderStream(t *testing.T) {
	p := NewPlugin()
	err := p.ExecuteProviderStream(context.Background(), "env", nil, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "streaming")
}
