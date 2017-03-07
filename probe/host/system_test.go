package host_test

import (
	"testing"
	"time"

	"github.com/weaveworks/scope/probe/host"
)

func TestGetLoad(t *testing.T) {
	have := host.GetLoad(time.Now())
	if len(have) != 1 {
		t.Fatalf("Expected 1 metrics, but got: %v", have)
	}
	for key, metric := range have {
		if metric.Len() != 1 {
			t.Errorf("Expected metric %v to have 1 sample, but had: %d", key, metric.Len())
		}
	}
}

func TestGetUptime(t *testing.T) {
	have, err := host.GetUptime()
	if err != nil {
		t.Fatal(err)
	}
	if have == 0 {
		t.Fatal(have)
	}
	t.Log(have.String())
}
