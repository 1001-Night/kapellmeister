package cmd

import (
	"fmt"

	"helm.sh/helm/v3/pkg/chartutil"
)

func loadValues(path string) (map[string]interface{}, error) {
	if path == "" {
		return map[string]interface{}{}, nil
	}

	values, err := chartutil.ReadValuesFile(path)
	if err != nil {
		return nil, fmt.Errorf("load values file %q: %w", path, err)
	}

	return values.AsMap(), nil
}
