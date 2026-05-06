package builtin

import (
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/require"
	"sigs.k8s.io/yaml"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/promotion"
	"github.com/akuity/kargo/pkg/x/promotion/runner/builtin"
)

func Test_yamlWriter_convert(t *testing.T) {
	tests := []validationTestCase{
		{
			name:   "path is not specified",
			config: promotion.Config{},
			expectedProblems: []string{
				"(root): path is required",
			},
		},
		{
			name: "path is empty",
			config: promotion.Config{
				"path": "",
			},
			expectedProblems: []string{
				"path: String length must be greater than or equal to 1",
			},
		},
		{
			name: "data is not specified",
			config: promotion.Config{
				"path": "config.yaml",
			},
			expectedProblems: []string{
				"(root): data is required",
			},
		},
		{
			name: "valid config with map data",
			config: promotion.Config{
				"path": "config.yaml",
				"data": map[string]any{
					"key": "value",
				},
			},
		},
		{
			name: "valid config with array data",
			config: promotion.Config{
				"path": "config.yaml",
				"data": []any{"one", 2, true},
			},
		},
		{
			name: "valid config with scalar data",
			config: promotion.Config{
				"path": "value.yaml",
				"data": "scalar",
			},
		},
	}

	r := newYAMLWriter(promotion.StepRunnerCapabilities{})
	runner, ok := r.(*yamlWriter)
	require.True(t, ok)

	runValidationTests(t, runner.convert, tests)
}

func Test_yamlWriter_run(t *testing.T) {
	tests := []struct {
		name       string
		setup      func(*testing.T, string)
		cfg        builtin.YAMLWriteConfig
		assertions func(*testing.T, string, promotion.StepResult, error)
	}{
		{
			name: "writes scalar",
			cfg: builtin.YAMLWriteConfig{
				Path: "value.yaml",
				Data: "fake-value",
			},
			assertions: func(t *testing.T, workDir string, result promotion.StepResult, err error) {
				require.NoError(t, err)
				require.Equal(t, promotion.StepResult{
					Status: kargoapi.PromotionStepStatusSucceeded,
					Output: map[string]any{
						"commitMessage": "Wrote value.yaml",
					},
				}, result)
				content, err := os.ReadFile(path.Join(workDir, "value.yaml"))
				require.NoError(t, err)
				require.Equal(t, "fake-value\n", string(content))
			},
		},
		{
			name: "writes map to nested directory",
			cfg: builtin.YAMLWriteConfig{
				Path: "out/config.yaml",
				Data: map[string]any{
					"configFiles": []any{
						"values/base.yaml",
						"values/production.yaml",
					},
					"deployment": map[string]any{
						"strategy": map[string]any{
							"retry": map[string]any{
								"backoff": map[string]any{
									"duration":    "15s",
									"factor":      3,
									"maxDuration": "3m",
								},
								"limit": 10,
							},
							"options": []any{},
						},
					},
				},
			},
			assertions: func(t *testing.T, workDir string, result promotion.StepResult, err error) {
				require.NoError(t, err)
				require.Equal(t, kargoapi.PromotionStepStatusSucceeded, result.Status)
				content, err := os.ReadFile(path.Join(workDir, "out/config.yaml"))
				require.NoError(t, err)
				assertYAMLEq(t, `configFiles:
- values/base.yaml
- values/production.yaml
deployment:
  strategy:
    retry:
      backoff:
        duration: 15s
        factor: 3
        maxDuration: 3m
      limit: 10
    options: []
`, string(content))
				require.True(t, len(content) > 0 && content[len(content)-1] == '\n',
					"output should end with a trailing newline")
			},
		},
		{
			name: "writes array",
			cfg: builtin.YAMLWriteConfig{
				Path: "array.yaml",
				Data: []any{"one", map[string]any{"two": true}},
			},
			assertions: func(t *testing.T, workDir string, result promotion.StepResult, err error) {
				require.NoError(t, err)
				require.Equal(t, kargoapi.PromotionStepStatusSucceeded, result.Status)
				content, err := os.ReadFile(path.Join(workDir, "array.yaml"))
				require.NoError(t, err)
				assertYAMLEq(t, `
- one
- two: true
`, string(content))
			},
		},
		{
			name: "overwrites existing file",
			setup: func(t *testing.T, workDir string) {
				require.NoError(t, os.WriteFile(
					path.Join(workDir, "config.yaml"),
					[]byte("old: true\nremoved: yes\n"),
					0o600,
				))
			},
			cfg: builtin.YAMLWriteConfig{
				Path: "config.yaml",
				Data: map[string]any{"new": "value"},
			},
			assertions: func(t *testing.T, workDir string, result promotion.StepResult, err error) {
				require.NoError(t, err)
				require.Equal(t, kargoapi.PromotionStepStatusSucceeded, result.Status)
				content, err := os.ReadFile(path.Join(workDir, "config.yaml"))
				require.NoError(t, err)
				require.NotContains(t, string(content), "old")
				require.NotContains(t, string(content), "removed")
				assertYAMLEq(t, "new: value\n", string(content))
			},
		},
		{
			name: "rejects relative parent traversal",
			cfg: builtin.YAMLWriteConfig{
				Path: "../escape.yaml",
				Data: "fake-value",
			},
			assertions: func(t *testing.T, _ string, result promotion.StepResult, err error) {
				require.ErrorContains(t, err, "attempts to traverse outside the working directory")
				require.Equal(t, kargoapi.PromotionStepStatusErrored, result.Status)
			},
		},
		{
			name: "rejects bare parent path",
			cfg: builtin.YAMLWriteConfig{
				Path: "..",
				Data: "fake-value",
			},
			assertions: func(t *testing.T, _ string, result promotion.StepResult, err error) {
				require.ErrorContains(t, err, "attempts to traverse outside the working directory")
				require.Equal(t, kargoapi.PromotionStepStatusErrored, result.Status)
			},
		},
		{
			name: "rejects deep traversal that resolves outside workdir",
			cfg: builtin.YAMLWriteConfig{
				Path: "subdir/../../escape.yaml",
				Data: "fake-value",
			},
			assertions: func(t *testing.T, _ string, result promotion.StepResult, err error) {
				require.ErrorContains(t, err, "attempts to traverse outside the working directory")
				require.Equal(t, kargoapi.PromotionStepStatusErrored, result.Status)
			},
		},
		{
			name: "rejects absolute path",
			cfg: builtin.YAMLWriteConfig{
				Path: "/etc/escape.yaml",
				Data: "fake-value",
			},
			assertions: func(t *testing.T, _ string, result promotion.StepResult, err error) {
				require.ErrorContains(t, err, "must be relative")
				require.Equal(t, kargoapi.PromotionStepStatusErrored, result.Status)
			},
		},
	}

	runner := &yamlWriter{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stepCtx := &promotion.StepContext{WorkDir: t.TempDir()}
			if tt.setup != nil {
				tt.setup(t, stepCtx.WorkDir)
			}
			result, err := runner.run(stepCtx, tt.cfg)
			tt.assertions(t, stepCtx.WorkDir, result, err)
		})
	}
}

func assertYAMLEq(t *testing.T, expected string, actual string) {
	t.Helper()

	var expectedData any
	require.NoError(t, yaml.Unmarshal([]byte(expected), &expectedData))
	var actualData any
	require.NoError(t, yaml.Unmarshal([]byte(actual), &actualData))
	require.Equal(t, expectedData, actualData)
}
