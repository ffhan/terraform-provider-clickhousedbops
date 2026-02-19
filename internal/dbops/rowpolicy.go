package dbops

import (
	"context"
	"fmt"
	"strings"

	"github.com/pingcap/errors"

	"github.com/ClickHouse/terraform-provider-clickhousedbops/internal/clickhouseclient"
	"github.com/ClickHouse/terraform-provider-clickhousedbops/internal/querybuilder"
)

type RowPolicy struct {
	Name            string
	Database        string
	Table           string
	SelectFilter    string
	IsRestrictive   bool
	GranteeUserName *string
	GranteeRoleName *string
}

func (i *impl) CreateRowPolicy(ctx context.Context, rp RowPolicy, clusterName *string) (*RowPolicy, error) {
	var to string
	if rp.GranteeUserName != nil {
		to = *rp.GranteeUserName
	} else if rp.GranteeRoleName != nil {
		to = *rp.GranteeRoleName
	} else {
		return nil, errors.New("either GranteeUserName or GranteeRoleName must be set")
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "CREATE ROW POLICY `%s`", rp.Name)

	if clusterName != nil && *clusterName != "" {
		fmt.Fprintf(&sb, " ON CLUSTER `%s`", *clusterName)
	}

	fmt.Fprintf(&sb, " ON `%s`.`%s`", rp.Database, rp.Table)
	fmt.Fprintf(&sb, " FOR SELECT USING %s", rp.SelectFilter)

	if rp.IsRestrictive {
		sb.WriteString(" AS RESTRICTIVE")
	} else {
		sb.WriteString(" AS PERMISSIVE")
	}

	fmt.Fprintf(&sb, " TO `%s`", to)

	err := i.clickhouseClient.Exec(ctx, sb.String())
	if err != nil {
		return nil, errors.WithMessage(err, "error running query")
	}

	identifier := fmt.Sprintf("%s ON %s.%s", rp.Name, rp.Database, rp.Table)

	return retryWithBackoff(ctx, "row policy", identifier, func() (*RowPolicy, error) {
		return i.GetRowPolicy(ctx, &rp, clusterName)
	})
}

func (i *impl) GetRowPolicy(ctx context.Context, rp *RowPolicy, clusterName *string) (*RowPolicy, error) {
	where := []querybuilder.Where{
		querybuilder.WhereEquals("short_name", rp.Name),
		querybuilder.WhereEquals("database", rp.Database),
		querybuilder.WhereEquals("table", rp.Table),
	}

	sql, err := querybuilder.NewSelect(
		[]querybuilder.Field{
			querybuilder.NewField("short_name"),
			querybuilder.NewField("select_filter"),
			querybuilder.NewField("is_restrictive"),
			querybuilder.NewField("grantee_name"),
		},
		"system.row_policies",
	).WithCluster(clusterName).Where(where...).Build()
	if err != nil {
		return nil, errors.WithMessage(err, "error building query")
	}

	var result *RowPolicy
	err = i.clickhouseClient.Select(ctx, sql, func(data clickhouseclient.Row) error {
		name, err := data.GetString("short_name")
		if err != nil {
			return errors.WithMessage(err, "error scanning query result, missing 'short_name' field")
		}

		selectFilter, err := data.GetString("select_filter")
		if err != nil {
			return errors.WithMessage(err, "error scanning query result, missing 'select_filter' field")
		}

		isRestrictive, err := data.GetBool("is_restrictive")
		if err != nil {
			return errors.WithMessage(err, "error scanning query result, missing 'is_restrictive' field")
		}

		granteeName, err := data.GetNullableString("grantee_name")
		if err != nil {
			return errors.WithMessage(err, "error scanning query result, missing 'grantee_name' field")
		}

		result = &RowPolicy{
			Name:          name,
			Database:      rp.Database,
			Table:         rp.Table,
			SelectFilter:  selectFilter,
			IsRestrictive: isRestrictive,
		}

		// Determine if grantee is user or role based on what we find
		if granteeName != nil {
			// In ClickHouse, we'd need to query system.users and system.roles to determine which type
			// For now, we'll preserve the original grantee assignment if provided, or try to infer
			if rp.GranteeUserName != nil && *rp.GranteeUserName == *granteeName {
				result.GranteeUserName = rp.GranteeUserName
			} else if rp.GranteeRoleName != nil && *rp.GranteeRoleName == *granteeName {
				result.GranteeRoleName = rp.GranteeRoleName
			} else {
				// Try to infer by checking if name exists in users or roles
				userExists, _ := i.userExists(ctx, *granteeName, clusterName)
				if userExists {
					result.GranteeUserName = granteeName
				} else {
					result.GranteeRoleName = granteeName
				}
			}
		}

		return nil
	})
	if err != nil {
		return nil, errors.WithMessage(err, "error running query")
	}

	return result, nil
}

func (i *impl) userExists(ctx context.Context, userName string, clusterName *string) (bool, error) {
	sql, err := querybuilder.NewSelect(
		[]querybuilder.Field{querybuilder.NewField("name")},
		"system.users",
	).WithCluster(clusterName).Where(querybuilder.WhereEquals("name", userName)).Build()
	if err != nil {
		return false, err
	}

	exists := false
	err = i.clickhouseClient.Select(ctx, sql, func(data clickhouseclient.Row) error {
		exists = true
		return nil
	})
	return exists, err
}

func (i *impl) UpdateRowPolicy(ctx context.Context, rp RowPolicy, clusterName *string) (*RowPolicy, error) {
	// Retrieve current row policy
	existing, err := i.GetRowPolicy(ctx, &rp, clusterName)
	if err != nil {
		return nil, errors.WithMessage(err, "Unable to get existing row policy")
	}

	if existing == nil {
		return nil, errors.New("row policy not found")
	}

	builder := querybuilder.NewAlterRowPolicy(rp.Name, rp.Database, rp.Table)

	if clusterName != nil && *clusterName != "" {
		builder = builder.WithCluster(clusterName)
	}

	// Only include changes in the ALTER statement
	if rp.SelectFilter != existing.SelectFilter {
		builder = builder.SelectFilter(rp.SelectFilter)
	}

	if rp.IsRestrictive != existing.IsRestrictive {
		builder = builder.IsRestrictive(rp.IsRestrictive)
	}

	sql, err := builder.Build()
	if err != nil {
		return nil, errors.WithMessage(err, "error building query")
	}

	err = i.clickhouseClient.Exec(ctx, sql)
	if err != nil {
		return nil, errors.WithMessage(err, "error running query")
	}

	identifier := fmt.Sprintf("%s ON %s.%s", rp.Name, rp.Database, rp.Table)

	return retryWithBackoff(ctx, "row policy", identifier, func() (*RowPolicy, error) {
		return i.GetRowPolicy(ctx, &rp, clusterName)
	})
}

func (i *impl) DeleteRowPolicy(ctx context.Context, name string, database string, table string, clusterName *string) error {
	var sb strings.Builder
	fmt.Fprintf(&sb, "DROP ROW POLICY IF EXISTS `%s`", name)

	if clusterName != nil && *clusterName != "" {
		fmt.Fprintf(&sb, " ON CLUSTER `%s`", *clusterName)
	}

	fmt.Fprintf(&sb, " ON `%s`.`%s`", database, table)

	err := i.clickhouseClient.Exec(ctx, sb.String())
	if err != nil {
		return errors.WithMessage(err, "error running query")
	}

	return nil
}
