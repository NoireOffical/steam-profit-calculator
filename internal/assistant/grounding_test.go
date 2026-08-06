package assistant

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestEnforceGroundingAcceptsCitedToolNumbers(t *testing.T) {
	evidence := []Evidence{{
		ID: "T1", Verified: true,
		Data: json.RawMessage(`{"evidence_id":"T1","records":[{"item_name":"Capsule","total_profit":0.948,"quantity":5}]}`),
	}}
	result := enforceGrounding("5 件 Capsule 的总收益为 0.948 [T1]。仅供研究。", evidence)
	if !result.Grounded {
		t.Fatalf("Grounded = false, note = %q", result.Note)
	}
}

func TestEnforceGroundingRejectsMissingCitation(t *testing.T) {
	evidence := []Evidence{{ID: "T1", Verified: true, Data: json.RawMessage(`{"evidence_id":"T1","total_profit":0.948}`)}}
	result := enforceGrounding("总收益为 0.948。", evidence)
	if result.Grounded || !strings.Contains(result.Text, "未通过数值溯源") {
		t.Fatalf("result = %#v, want fallback", result)
	}
}

func TestEnforceGroundingRejectsInventedNumber(t *testing.T) {
	evidence := []Evidence{{ID: "T1", Verified: true, Data: json.RawMessage(`{"evidence_id":"T1","total_profit":0.948}`)}}
	result := enforceGrounding("总收益为 9.999 [T1]。", evidence)
	if result.Grounded || !strings.Contains(result.Note, "absent") {
		t.Fatalf("result = %#v, want unsupported-number fallback", result)
	}
}

func TestEnforceGroundingRejectsNumbersWithoutToolEvidence(t *testing.T) {
	result := enforceGrounding("建议卖到 0.800。", nil)
	if result.Grounded {
		t.Fatal("Grounded = true, want false")
	}
}

func TestEnforceGroundingAllowsNonNumericClarification(t *testing.T) {
	result := enforceGrounding("请补充买入价和目标卖价。", nil)
	if !result.Grounded || result.Text == "" {
		t.Fatalf("result = %#v, want unchanged grounded clarification", result)
	}
}
