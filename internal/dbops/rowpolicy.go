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
		},
		"system.row_policies",
	).WithCluster(clusterName).Where(where...).Build()
	if err != nil {
		return nil, errors.WithMessage(err, "error building query")
	}

	found := false
	err = i.clickhouseClient.Select(ctx, sql, func(data clickhouseclient.Row) error {
		_, err := data.GetString("short_name")
		if err != nil {
			return errors.WithMessage(err, "error scanning query result, missing 'short_name' field")
		}
		found = true
		return nil
	})
	if err != nil {
		return nil, errors.WithMessage(err, "error running query")
	}

	if !found {
		return nil, nil
	}

	return rp, nil
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
