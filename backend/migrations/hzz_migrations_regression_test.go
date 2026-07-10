package migrations

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCustomizedMigrationChecksumsRemainStable(t *testing.T) {
	expected := map[string]string{
		"140_add_group_scheduling_strategy.sql":          "01d8d903e21056e09acf84664bbd7a7ee6ff99b414101489f4fbe7e7d03457ee",
		"145_hzz_agent_promotion.sql":                    "9bab8fd95ef6497ffcae006d8df67bb0da224533733babd838331538d7204988",
		"146_hzz_agent_group_delegations.sql":            "8bc9a0255f3451c6cd21e99d7ba753feb57051b108213aa8d5438c9f7dbec590",
		"147_hzz_agent_profiles.sql":                     "968f288962446ae39cb9aef38975aacd5cc1a6ee894dbddba08f4346e56e56ec",
		"148_hzz_agent_invite_defaults.sql":              "1c8764dd7eb37ed85530a9d30757535b999418e3c33abc8bd174ea306520c5ff",
		"149_hzz_enterprise_management.sql":              "1781604c9e66a162d9d842c40cf096a21ed08a38064254203c1d6c1096f4f1a7",
		"150_hzz_enterprise_balance_logs_audit.sql":      "48cb37178427c24ccfdd267f8bae5cd6ee780e4790dd1f623e455ad75741cd68",
		"151_hzz_agent_invite_group_defaults.sql":        "68e1c76b8aafb23b505750e8b1d5e9e57901375e8e581d927874d83d8947b7a6",
		"152_hzz_purchase_info_cards.sql":                "05c71aa689867941bd26a1fffa499cfb4f9830d18a397f871c704321d77f9f22",
		"153_hzz_agent_income_snapshots.sql":             "71067d68f76521f7a979b080183a3979780394598d7fb30dfe5bf01982bc1f85",
		"154_hzz_promote_level2_agents.sql":              "1d96cd6c9d562c005486d0d0ce11ac266cbdbf4c5f94a044f8c389aa840e2e0f",
		"155_hzz_agent_income_adjustments.sql":           "3f09555f407d5943687a121550ca3c061a1918a04967c7989036d76d9032c95d",
		"156_hzz_enterprise_employee_group_defaults.sql": "988341cb2a64119c9cfe586232ce16ac7ecbb2272de65eafc2c8a1ed79bf3e13",
		"157_hzz_agent_child_notes.sql":                  "7f9f91dd42b8696d3b083128e474bbd32ba8cb2a87237d6758a2eae06b1f9444",
		"158_hzz_scheduled_test_auto_schedulable.sql":    "691944ae5f172719ad596f9503ca2ca623b29c04179fd68bc6cba8fd7d78f3a5",
	}

	for name, want := range expected {
		content, err := FS.ReadFile(name)
		require.NoError(t, err, name)
		sum := sha256.Sum256([]byte(strings.TrimSpace(string(content))))
		require.Equal(t, want, hex.EncodeToString(sum[:]), name)
	}
}
