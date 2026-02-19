package querybuilder

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAlterRowPolicy_Basic(t *testing.T) {
	tests := []struct {
		name    string
		builder *AlterRowPolicy
		want    string
		wantErr bool
	}{
		{
			name: "alter row policy select filter only",
			builder: NewAlterRowPolicy("my_policy", "default", "users").
				SelectFilter("user_id = 'alice'"),
			want: "ALTER ROW POLICY `my_policy` ON `default`.`users` FOR SELECT USING user_id = 'alice'",
		},
		{
			name: "alter row policy is_restrictive only",
			builder: NewAlterRowPolicy("my_policy", "default", "users").
				IsRestrictive(true),
			want: "ALTER ROW POLICY `my_policy` ON `default`.`users` FOR SELECT AS RESTRICTIVE",
		},
		{
			name: "alter row policy both select filter and is_restrictive",
			builder: NewAlterRowPolicy("my_policy", "default", "users").
				SelectFilter("1").
				IsRestrictive(false),
			want: "ALTER ROW POLICY `my_policy` ON `default`.`users` FOR SELECT USING 1 AS PERMISSIVE",
		},
		{
			name: "alter row policy with cluster",
			builder: NewAlterRowPolicy("my_policy", "default", "users").
				WithCluster(stringPtr("cluster1")).
				SelectFilter("tenant_id = 'abc'"),
			want: "ALTER ROW POLICY `my_policy` ON CLUSTER `cluster1` ON `default`.`users` FOR SELECT USING tenant_id = 'abc'",
		},
		{
			name:    "alter row policy no changes",
			builder: NewAlterRowPolicy("my_policy", "default", "users"),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.builder.Build()
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func stringPtr(s string) *string {
	return &s
}
