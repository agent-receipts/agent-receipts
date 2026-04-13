package policy

import "testing"

func TestEvaluatePass(t *testing.T) {
	engine := NewEngine(DefaultRules())
	d := engine.Evaluate(EvalContext{
		ToolName:      "read_file",
		ServerName:    "filesystem",
		OperationType: "read",
		RiskScore:     0,
	})
	if d.Action != "pass" {
		t.Errorf("expected pass, got %s", d.Action)
	}
}

func TestEvaluateBlock(t *testing.T) {
	engine := NewEngine(DefaultRules())
	d := engine.Evaluate(EvalContext{
		ToolName:      "delete_secrets",
		ServerName:    "vault",
		OperationType: "delete",
		RiskScore:     80,
	})
	if d.Action != "block" {
		t.Errorf("expected block, got %s (rule: %s)", d.Action, d.RuleName)
	}
}

func TestEvaluatePause(t *testing.T) {
	engine := NewEngine(DefaultRules())
	d := engine.Evaluate(EvalContext{
		ToolName:      "update_auth_config",
		ServerName:    "settings",
		OperationType: "write",
		RiskScore:     60,
	})
	if d.Action != "pause" {
		t.Errorf("expected pause, got %s (rule: %s)", d.Action, d.RuleName)
	}
}

func TestEvaluateFlag(t *testing.T) {
	engine := NewEngine(DefaultRules())
	d := engine.Evaluate(EvalContext{
		ToolName:      "send_message",
		ServerName:    "slack",
		OperationType: "write",
		RiskScore:     20,
	})
	if d.Action != "flag" {
		t.Errorf("expected flag, got %s (rule: %s)", d.Action, d.RuleName)
	}
}

func TestMostRestrictiveWins(t *testing.T) {
	risk30 := 30
	rules := []Rule{
		{Name: "flag_all", Enabled: true, Action: "flag"},
		{Name: "pause_risky", Enabled: true, MinRiskScore: &risk30, Action: "pause"},
	}
	engine := NewEngine(rules)
	d := engine.Evaluate(EvalContext{RiskScore: 50})
	if d.Action != "pause" {
		t.Errorf("expected pause (most restrictive), got %s", d.Action)
	}
}

func TestEmptyRulesPass(t *testing.T) {
	engine := NewEngine([]Rule{})
	d := engine.Evaluate(EvalContext{
		ToolName:  "anything",
		RiskScore: 100,
	})
	if d.Action != "pass" {
		t.Errorf("expected pass with no rules, got %s", d.Action)
	}
}

func TestCombinedToolPatternAndRiskScore(t *testing.T) {
	risk40 := 40
	rules := []Rule{
		{
			Name:         "block_dangerous_delete",
			Enabled:      true,
			ToolPattern:  "delete_*",
			MinRiskScore: &risk40,
			Action:       "block",
		},
	}
	engine := NewEngine(rules)

	// Matches both tool pattern and risk score — should block.
	d := engine.Evaluate(EvalContext{ToolName: "delete_files", RiskScore: 50})
	if d.Action != "block" {
		t.Errorf("expected block, got %s", d.Action)
	}

	// Matches tool pattern but NOT risk score — should pass.
	d = engine.Evaluate(EvalContext{ToolName: "delete_files", RiskScore: 30})
	if d.Action != "pass" {
		t.Errorf("expected pass (risk below threshold), got %s", d.Action)
	}

	// Matches risk score but NOT tool pattern — should pass.
	d = engine.Evaluate(EvalContext{ToolName: "read_file", RiskScore: 50})
	if d.Action != "pass" {
		t.Errorf("expected pass (tool pattern mismatch), got %s", d.Action)
	}
}

func TestDisabledRulesSkipped(t *testing.T) {
	rules := []Rule{
		{Name: "block_all", Enabled: false, Action: "block"},
	}
	engine := NewEngine(rules)
	d := engine.Evaluate(EvalContext{})
	if d.Action != "pass" {
		t.Errorf("expected pass (rule disabled), got %s", d.Action)
	}
}

func TestHasPauseRules(t *testing.T) {
	// Default rules include pause_high_risk.
	engine := NewEngine(DefaultRules())
	if !engine.HasPauseRules() {
		t.Error("expected HasPauseRules=true for default rules")
	}

	// Flag-only rules should not require the approval server.
	engine = NewEngine([]Rule{
		{Name: "flag_only", Enabled: true, Action: "flag"},
	})
	if engine.HasPauseRules() {
		t.Error("expected HasPauseRules=false for flag-only rules")
	}

	// Empty rules.
	engine = NewEngine([]Rule{})
	if engine.HasPauseRules() {
		t.Error("expected HasPauseRules=false for empty rules")
	}

	// Disabled pause rule should not count.
	engine = NewEngine([]Rule{
		{Name: "disabled_pause", Enabled: false, Action: "pause"},
	})
	if engine.HasPauseRules() {
		t.Error("expected HasPauseRules=false when pause rule is disabled")
	}
}
