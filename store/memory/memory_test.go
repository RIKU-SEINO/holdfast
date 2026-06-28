package memory_test

import (
	"testing"

	"github.com/RIKU-SEINO/holdfast/conformance"
	"github.com/RIKU-SEINO/holdfast/store/memory"
)

func TestMemoryStore(t *testing.T) {
	conformance.Run(t, memory.New())
}
