package modulessdk

import "testing"

func TestDefaultQueryConfigParses(t *testing.T) {
	if len(DefaultJoinerConfigs) != 53 || len(DefaultRelationMap) == 0 {
		t.Fatalf("unexpected config sizes: %d configs, %d relation map entities", len(DefaultJoinerConfigs), len(DefaultRelationMap))
	}
}
