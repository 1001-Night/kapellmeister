package helm

import (
	"fmt"
	"io"
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/yaml"
)

// разбор манифеста
func manifestObjects(manifest string) ([]unstructured.Unstructured, error) {

	decoder := yaml.NewYAMLOrJSONDecoder(strings.NewReader(manifest), 4096)
	var objects []unstructured.Unstructured

	for {
		var object unstructured.Unstructured
		err := decoder.Decode(&object)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("decode manifest: %w", err)
		}
		if len(object.Object) == 0 {
			continue
		}
		objects = append(objects, object)
	}

	return objects, nil
}
