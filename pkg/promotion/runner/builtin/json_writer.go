package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	securejoin "github.com/cyphar/filepath-securejoin"
	"github.com/xeipuuv/gojsonschema"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/promotion"
	"github.com/akuity/kargo/pkg/x/promotion/runner/builtin"
)

const stepKindJSONWrite = "json-write"

func init() {
	promotion.DefaultStepRunnerRegistry.MustRegister(
		promotion.StepRunnerRegistration{
			Name:  stepKindJSONWrite,
			Value: newJSONWriter,
		},
	)
}

// jsonWriter is an implementation of the promotion.StepRunner interface that
// writes arbitrary data to a JSON file.
type jsonWriter struct {
	schemaLoader gojsonschema.JSONLoader
}

// newJSONWriter returns an implementation of the promotion.StepRunner interface
// that writes arbitrary data to a JSON file.
func newJSONWriter(promotion.StepRunnerCapabilities) promotion.StepRunner {
	return &jsonWriter{schemaLoader: getConfigSchemaLoader(stepKindJSONWrite)}
}

// Run implements the promotion.StepRunner interface.
func (j *jsonWriter) Run(
	_ context.Context,
	stepCtx *promotion.StepContext,
) (promotion.StepResult, error) {
	cfg, err := j.convert(stepCtx.Config)
	if err != nil {
		return promotion.StepResult{
			Status: kargoapi.PromotionStepStatusFailed,
		}, &promotion.TerminalError{Err: err}
	}
	return j.run(stepCtx, cfg)
}

func (j *jsonWriter) convert(cfg promotion.Config) (builtin.JSONWriteConfig, error) {
	return validateAndConvert[builtin.JSONWriteConfig](j.schemaLoader, cfg, stepKindJSONWrite)
}

func (j *jsonWriter) run(
	stepCtx *promotion.StepContext,
	cfg builtin.JSONWriteConfig,
) (promotion.StepResult, error) {
	absPath, err := secureJoinWritePath(stepCtx.WorkDir, cfg.Path)
	if err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
			fmt.Errorf("error joining path %q: %w", cfg.Path, err)
	}

	data, err := json.MarshalIndent(cfg.Data, "", "  ")
	if err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
			fmt.Errorf("error marshaling JSON data: %w", err)
	}
	data = append(data, '\n')

	if err = os.MkdirAll(filepath.Dir(absPath), 0o700); err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
			fmt.Errorf("error creating directory for %q: %w", cfg.Path, err)
	}
	if err = os.WriteFile(absPath, data, 0o600); err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
			fmt.Errorf("error writing JSON file %q: %w", cfg.Path, err)
	}

	return promotion.StepResult{
		Status: kargoapi.PromotionStepStatusSucceeded,
		Output: map[string]any{
			"commitMessage": fmt.Sprintf("Wrote %s", cfg.Path),
		},
	}, nil
}

func secureJoinWritePath(workDir string, path string) (string, error) {
	if filepath.IsAbs(path) {
		return "", fmt.Errorf("path %q must be relative", path)
	}
	cleanPath := filepath.Clean(path)
	if cleanPath == ".." || strings.HasPrefix(cleanPath, fmt.Sprintf("..%c", os.PathSeparator)) {
		return "", fmt.Errorf("path %q attempts to traverse outside the working directory", path)
	}
	return securejoin.SecureJoin(workDir, path)
}
