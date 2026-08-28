package conformance_test

import (
	"testing"

	"m31labs.dev/gosx-studio/conformance"
)

func TestOperationProtocolConformance(t *testing.T) {
	conformance.RunOperationProtocolConformance(t)
}
