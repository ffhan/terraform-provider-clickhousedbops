package querybuilder

import (
	"fmt"
	"strings"
)

// AlterRowPolicy is a query builder for ALTER ROW POLICY statements.
type AlterRowPolicy struct {
	name          string
	database      string
	table         string
	clusterName   *string
	selectFilter  *string
	isRestrictive *bool
}

// NewAlterRowPolicy creates a new AlterRowPolicy builder.
func NewAlterRowPolicy(name string, database string, table string) *AlterRowPolicy {
	return &AlterRowPolicy{
		name:     name,
		database: database,
		table:    table,
	}
}

// WithCluster sets the cluster name for the ALTER statement.
func (a *AlterRowPolicy) WithCluster(clusterName *string) *AlterRowPolicy {
	a.clusterName = clusterName
	return a
}

// SelectFilter sets the select filter for the row policy.
func (a *AlterRowPolicy) SelectFilter(filter string) *AlterRowPolicy {
	a.selectFilter = &filter
	return a
}

// IsRestrictive sets whether the policy is restrictive or permissive.
func (a *AlterRowPolicy) IsRestrictive(restrictive bool) *AlterRowPolicy {
	a.isRestrictive = &restrictive
	return a
}

// Build generates the ALTER ROW POLICY SQL statement.
func (a *AlterRowPolicy) Build() (string, error) {
	var sb strings.Builder

	fmt.Fprintf(&sb, "ALTER ROW POLICY `%s`", a.name)

	if a.clusterName != nil && *a.clusterName != "" {
		fmt.Fprintf(&sb, " ON CLUSTER `%s`", *a.clusterName)
	}

	fmt.Fprintf(&sb, " ON `%s`.`%s`", a.database, a.table)

	// At least one modification is required
	hasChanges := false

	if a.selectFilter != nil {
		fmt.Fprintf(&sb, " FOR SELECT USING %s", *a.selectFilter)
		hasChanges = true
	}

	if a.isRestrictive != nil {
		if !hasChanges {
			sb.WriteString(" FOR SELECT")
		}
		if *a.isRestrictive {
			sb.WriteString(" AS RESTRICTIVE")
		} else {
			sb.WriteString(" AS PERMISSIVE")
		}
		hasChanges = true
	}

	if !hasChanges {
		return "", fmt.Errorf("at least one change must be specified for ALTER ROW POLICY")
	}

	return sb.String(), nil
}
