package builtin

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/xeipuuv/gojsonschema"
	"sigs.k8s.io/yaml"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/promotion"
	"github.com/akuity/kargo/pkg/x/promotion/runner/builtin"
)

const stepKindYAMLWrite = "yaml-write"

func init() {
	promotion.DefaultStepRunnerRegistry.MustRegister(
		promotion.StepRunnerRegistration{
			Name:  stepKindYAMLWrite,
			Value: newYAMLWriter,
		},
	)
}

// yamlWriter is an implementation of the promotion.StepRunner interface that
// writes arbitrary data to a YAML file.
type yamlWriter struct {
	schemaLoader gojsonschema.JSONLoader
}

// newYAMLWriter returns an implementation of the promotion.StepRunner interface
// that writes arbitrary data to a YAML file.
func newYAMLWriter(promotion.StepRunnerCapabilities) promotion.StepRunner {
	return &yamlWriter{schemaLoader: getConfigSchemaLoader(stepKindYAMLWrite)}
}

// Run implements the promotion.StepRunner interface.
func (y *yamlWriter) Run(
	_ context.Context,
	stepCtx *promotion.StepContext,
) (promotion.StepResult, error) {
	cfg, err := y.convert(stepCtx.Config)
	if err != nil {
		return promotion.StepResult{
			Status: kargoapi.PromotionStepStatusFailed,
		}, &promotion.TerminalError{Err: err}
	}
	return y.run(stepCtx, cfg)
}

func (y *yamlWriter) convert(cfg promotion.Config) (builtin.YAMLWriteConfig, error) {
	return validateAndConvert[builtin.YAMLWriteConfig](y.schemaLoader, cfg, stepKindYAMLWrite)
}

func (y *yamlWriter) run(
	stepCtx *promotion.StepContext,
	cfg builtin.YAMLWriteConfig,
) (promotion.StepResult, error) {
	absPath, err := secureJoinWritePath(stepCtx.WorkDir, cfg.Path)
	if err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
			fmt.Errorf("error joining path %q: %w", cfg.Path, err)
	}

	data, err := yaml.Marshal(cfg.Data)
	if err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
			fmt.Errorf("error marshaling YAML data: %w", err)
	}

	if err = os.MkdirAll(filepath.Dir(absPath), 0o700); err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
			fmt.Errorf("error creating directory for %q: %w", cfg.Path, err)
	}
	if err = os.WriteFile(absPath, data, 0o600); err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
			fmt.Errorf("error writing YAML file %q: %w", cfg.Path, err)
	}

	return promotion.StepResult{
		Status: kargoapi.PromotionStepStatusSucceeded,
		Output: map[string]any{
			"commitMessage": fmt.Sprintf("Wrote %s", cfg.Path),
		},
	}, nil
}
