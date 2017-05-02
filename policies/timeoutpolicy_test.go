package policies

import (
	"testing"
	"time"
)

func TestTimeoutPolicy(t *testing.T) {
	policy := UnknownPurposeTimeout
	result := func() error {
		time.Sleep(2 * time.Second)
		return nil
	}
	err := policy.ExecuteAction(result)
	if err != nil {
		t.Errorf("Action should ahve returned nil: %v", err)
	}
}

func TestTimeoutPolicyExpires(t *testing.T) {
	policy := TimeoutPolicy(1 * time.Second)
	result := func() error {
		time.Sleep(5 * time.Second)
		return nil
	}
	err := policy.ExecuteAction(result)
	if err == nil {
		t.Errorf("Action should have returned err")
	}
}
