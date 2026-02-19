package rowpolicy_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/ClickHouse/terraform-provider-clickhousedbops/internal/dbops"
	"github.com/ClickHouse/terraform-provider-clickhousedbops/internal/testutils/nilcompare"
	"github.com/ClickHouse/terraform-provider-clickhousedbops/internal/testutils/resourcebuilder"
	"github.com/ClickHouse/terraform-provider-clickhousedbops/internal/testutils/runner"
)

const (
	resourceType = "clickhousedbops_row_policy"
	resourceName = "foo"

	granteeRoleName = "grantee"
	granteeUserName = "user1"
)

func TestRowpolicy_acceptance(t *testing.T) {
	clusterName := "cluster1"

	granteeRoleResource := resourcebuilder.
		New("clickhousedbops_role", granteeRoleName).
		WithStringAttribute("name", granteeRoleName)
	granteeUserResource := resourcebuilder.
		New("clickhousedbops_user", granteeUserName).
		WithStringAttribute("name", granteeUserName).
		WithFunction("password_sha256_hash_wo", "sha256", "test").
		WithIntAttribute("password_sha256_hash_wo_version", 1)

	checkNotExistsFunc := func(ctx context.Context, dbopsClient dbops.Client, clusterName *string, attrs map[string]string) (bool, error) {
		name := attrs["name"]
		if name == "" {
			return false, fmt.Errorf("name attribute was not set")
		}

		database := attrs["database_name"]
		if database == "" {
			return false, fmt.Errorf("database_name attribute was not set")
		}

		table := attrs["table_name"]
		if table == "" {
			return false, fmt.Errorf("table_name attribute was not set")
		}

		granteeUser := attrs["grantee_user_name"]
		granteeRole := attrs["grantee_role_name"]

		if granteeUser == "" && granteeRole == "" {
			return false, fmt.Errorf("both grantee_user_name and grantee_role_name attribute were not set")
		}

		var granteeUserNamePtr, granteeRoleNamePtr *string
		if granteeUser != "" {
			granteeUserNamePtr = &granteeUser
		}
		if granteeRole != "" {
			granteeRoleNamePtr = &granteeRole
		}

		rowPolicy := dbops.RowPolicy{
			Name:            name,
			Database:        database,
			Table:           table,
			GranteeUserName: granteeUserNamePtr,
			GranteeRoleName: granteeRoleNamePtr,
		}

		rp, err := dbopsClient.GetRowPolicy(ctx, &rowPolicy, clusterName)
		return rp != nil, err
	}

	checkAttributesFunc := func(ctx context.Context, dbopsClient dbops.Client, clusterName *string, attrs map[string]interface{}) error {
		name := attrs["name"].(string)
		if name == "" {
			return fmt.Errorf("name attribute was not set")
		}

		database := attrs["database_name"].(string)
		if database == "" {
			return fmt.Errorf("database_name attribute was not set")
		}

		table := attrs["table_name"].(string)
		if table == "" {
			return fmt.Errorf("table_name attribute was not set")
		}

		var granteeUserNamePtr, granteeRoleNamePtr *string
		if attrs["grantee_user_name"] != nil {
			s := attrs["grantee_user_name"].(string)
			granteeUserNamePtr = &s
		}

		if attrs["grantee_role_name"] != nil {
			s := attrs["grantee_role_name"].(string)
			granteeRoleNamePtr = &s
		}

		if granteeUserNamePtr == nil && granteeRoleNamePtr == nil {
			return fmt.Errorf("both grantee_user_name and grantee_role_name attribute were not set")
		}

		isRestrictive := false
		if attrs["is_restrictive"] != nil {
			isRestrictive = attrs["is_restrictive"].(bool)
		}

		rowPolicy := dbops.RowPolicy{
			Name:            name,
			Database:        database,
			Table:           table,
			GranteeUserName: granteeUserNamePtr,
			GranteeRoleName: granteeRoleNamePtr,
		}

		rp, err := dbopsClient.GetRowPolicy(ctx, &rowPolicy, clusterName)
		if err != nil {
			return err
		}

		if rp == nil {
			return fmt.Errorf("row policy was not found")
		}

		if name != rp.Name {
			return fmt.Errorf("expected name to be %q, was %q", name, rp.Name)
		}

		if database != rp.Database {
			return fmt.Errorf("expected database_name to be %q, was %q", database, rp.Database)
		}

		if table != rp.Table {
			return fmt.Errorf("expected table_name to be %q, was %q", table, rp.Table)
		}

		if !nilcompare.NilCompare(clusterName, attrs["cluster_name"]) {
			return fmt.Errorf("wrong value for cluster_name attribute")
		}

		if !nilcompare.NilCompare(rp.GranteeUserName, attrs["grantee_user_name"]) {
			return fmt.Errorf("wrong value for grantee_user_name attribute")
		}

		if !nilcompare.NilCompare(rp.GranteeRoleName, attrs["grantee_role_name"]) {
			return fmt.Errorf("wrong value for grantee_role_name attribute")
		}

		if rp.IsRestrictive != isRestrictive {
			return fmt.Errorf("expected is_restrictive to be %v, was %v", isRestrictive, rp.IsRestrictive)
		}

		return nil
	}

	tests := []runner.TestCase{
		// Single replica, Native
		{
			Name:     "Create permissive row policy for role using Native protocol on a single replica",
			ChEnv:    map[string]string{"CONFIGFILE": "config-single.xml"},
			Protocol: "native",
			Resource: resourcebuilder.New(resourceType, resourceName).
				WithStringAttribute("name", "test_policy").
				WithStringAttribute("database_name", "system").
				WithStringAttribute("table_name", "databases").
				WithStringAttribute("select_filter", "name = 'default'").
				WithResourceFieldReference("grantee_role_name", "clickhousedbops_role", granteeRoleName, "name").
				AddDependency(granteeRoleResource.Build()).
				Build(),
			ResourceName:        resourceName,
			ResourceAddress:     fmt.Sprintf("%s.%s", resourceType, resourceName),
			CheckNotExistsFunc:  checkNotExistsFunc,
			CheckAttributesFunc: checkAttributesFunc,
		},
		{
			Name:     "Create permissive row policy for user using Native protocol on a single replica",
			ChEnv:    map[string]string{"CONFIGFILE": "config-single.xml"},
			Protocol: "native",
			Resource: resourcebuilder.New(resourceType, resourceName).
				WithStringAttribute("name", "test_policy").
				WithStringAttribute("database_name", "system").
				WithStringAttribute("table_name", "databases").
				WithStringAttribute("select_filter", "1").
				WithResourceFieldReference("grantee_user_name", "clickhousedbops_user", granteeUserName, "name").
				AddDependency(granteeUserResource.Build()).
				Build(),
			ResourceName:        resourceName,
			ResourceAddress:     fmt.Sprintf("%s.%s", resourceType, resourceName),
			CheckNotExistsFunc:  checkNotExistsFunc,
			CheckAttributesFunc: checkAttributesFunc,
		},
		{
			Name:     "Create restrictive row policy for user using Native protocol on a single replica",
			ChEnv:    map[string]string{"CONFIGFILE": "config-single.xml"},
			Protocol: "native",
			Resource: resourcebuilder.New(resourceType, resourceName).
				WithStringAttribute("name", "test_policy").
				WithStringAttribute("database_name", "system").
				WithStringAttribute("table_name", "databases").
				WithStringAttribute("select_filter", "name != 'system'").
				WithResourceFieldReference("grantee_user_name", "clickhousedbops_user", granteeUserName, "name").
				WithBoolAttribute("is_restrictive", true).
				AddDependency(granteeUserResource.Build()).
				Build(),
			ResourceName:        resourceName,
			ResourceAddress:     fmt.Sprintf("%s.%s", resourceType, resourceName),
			CheckNotExistsFunc:  checkNotExistsFunc,
			CheckAttributesFunc: checkAttributesFunc,
		},
		// Single replica, HTTP
		{
			Name:     "Create permissive row policy for role using HTTP protocol on a single replica",
			ChEnv:    map[string]string{"CONFIGFILE": "config-single.xml"},
			Protocol: "http",
			Resource: resourcebuilder.New(resourceType, resourceName).
				WithStringAttribute("name", "test_policy").
				WithStringAttribute("database_name", "system").
				WithStringAttribute("table_name", "tables").
				WithStringAttribute("select_filter", "database = 'default'").
				WithResourceFieldReference("grantee_role_name", "clickhousedbops_role", granteeRoleName, "name").
				AddDependency(granteeRoleResource.Build()).
				Build(),
			ResourceName:        resourceName,
			ResourceAddress:     fmt.Sprintf("%s.%s", resourceType, resourceName),
			CheckNotExistsFunc:  checkNotExistsFunc,
			CheckAttributesFunc: checkAttributesFunc,
		},
		{
			Name:     "Create permissive row policy for user using HTTP protocol on a single replica",
			ChEnv:    map[string]string{"CONFIGFILE": "config-single.xml"},
			Protocol: "http",
			Resource: resourcebuilder.New(resourceType, resourceName).
				WithStringAttribute("name", "test_policy").
				WithStringAttribute("database_name", "system").
				WithStringAttribute("table_name", "databases").
				WithStringAttribute("select_filter", "name = 'default'").
				WithResourceFieldReference("grantee_user_name", "clickhousedbops_user", granteeUserName, "name").
				AddDependency(granteeUserResource.Build()).
				Build(),
			ResourceName:        resourceName,
			ResourceAddress:     fmt.Sprintf("%s.%s", resourceType, resourceName),
			CheckNotExistsFunc:  checkNotExistsFunc,
			CheckAttributesFunc: checkAttributesFunc,
		},
		{
			Name:     "Create restrictive row policy for user using HTTP protocol on a single replica",
			ChEnv:    map[string]string{"CONFIGFILE": "config-single.xml"},
			Protocol: "http",
			Resource: resourcebuilder.New(resourceType, resourceName).
				WithStringAttribute("name", "test_policy").
				WithStringAttribute("database_name", "system").
				WithStringAttribute("table_name", "databases").
				WithStringAttribute("select_filter", "1").
				WithResourceFieldReference("grantee_user_name", "clickhousedbops_user", granteeUserName, "name").
				WithBoolAttribute("is_restrictive", true).
				AddDependency(granteeUserResource.Build()).
				Build(),
			ResourceName:        resourceName,
			ResourceAddress:     fmt.Sprintf("%s.%s", resourceType, resourceName),
			CheckNotExistsFunc:  checkNotExistsFunc,
			CheckAttributesFunc: checkAttributesFunc,
		},
		// Replicated storage, native
		{
			Name:     "Create permissive row policy for role using Native protocol on a cluster using replicated storage",
			ChEnv:    map[string]string{"CONFIGFILE": "config-replicated.xml"},
			Protocol: "native",
			Resource: resourcebuilder.New(resourceType, resourceName).
				WithStringAttribute("name", "test_policy").
				WithStringAttribute("database_name", "system").
				WithStringAttribute("table_name", "databases").
				WithStringAttribute("select_filter", "name = 'default'").
				WithResourceFieldReference("grantee_role_name", "clickhousedbops_role", granteeRoleName, "name").
				AddDependency(granteeRoleResource.Build()).
				Build(),
			ResourceName:        resourceName,
			ResourceAddress:     fmt.Sprintf("%s.%s", resourceType, resourceName),
			CheckNotExistsFunc:  checkNotExistsFunc,
			CheckAttributesFunc: checkAttributesFunc,
		},
		{
			Name:     "Create permissive row policy for user using Native protocol on a cluster using replicated storage",
			ChEnv:    map[string]string{"CONFIGFILE": "config-replicated.xml"},
			Protocol: "native",
			Resource: resourcebuilder.New(resourceType, resourceName).
				WithStringAttribute("name", "test_policy").
				WithStringAttribute("database_name", "system").
				WithStringAttribute("table_name", "tables").
				WithStringAttribute("select_filter", "database = 'system'").
				WithResourceFieldReference("grantee_user_name", "clickhousedbops_user", granteeUserName, "name").
				AddDependency(granteeUserResource.Build()).
				Build(),
			ResourceName:        resourceName,
			ResourceAddress:     fmt.Sprintf("%s.%s", resourceType, resourceName),
			CheckNotExistsFunc:  checkNotExistsFunc,
			CheckAttributesFunc: checkAttributesFunc,
		},
		{
			Name:     "Create restrictive row policy for user using Native protocol on a cluster using replicated storage",
			ChEnv:    map[string]string{"CONFIGFILE": "config-replicated.xml"},
			Protocol: "native",
			Resource: resourcebuilder.New(resourceType, resourceName).
				WithStringAttribute("name", "test_policy").
				WithStringAttribute("database_name", "system").
				WithStringAttribute("table_name", "databases").
				WithStringAttribute("select_filter", "name != 'system'").
				WithResourceFieldReference("grantee_user_name", "clickhousedbops_user", granteeUserName, "name").
				WithBoolAttribute("is_restrictive", true).
				AddDependency(granteeUserResource.Build()).
				Build(),
			ResourceName:        resourceName,
			ResourceAddress:     fmt.Sprintf("%s.%s", resourceType, resourceName),
			CheckNotExistsFunc:  checkNotExistsFunc,
			CheckAttributesFunc: checkAttributesFunc,
		},
		// Replicated storage, http
		{
			Name:     "Create permissive row policy for role using HTTP protocol on a cluster using replicated storage",
			ChEnv:    map[string]string{"CONFIGFILE": "config-replicated.xml"},
			Protocol: "http",
			Resource: resourcebuilder.New(resourceType, resourceName).
				WithStringAttribute("name", "test_policy").
				WithStringAttribute("database_name", "system").
				WithStringAttribute("table_name", "tables").
				WithStringAttribute("select_filter", "database = 'default'").
				WithResourceFieldReference("grantee_role_name", "clickhousedbops_role", granteeRoleName, "name").
				AddDependency(granteeRoleResource.Build()).
				Build(),
			ResourceName:        resourceName,
			ResourceAddress:     fmt.Sprintf("%s.%s", resourceType, resourceName),
			CheckNotExistsFunc:  checkNotExistsFunc,
			CheckAttributesFunc: checkAttributesFunc,
		},
		{
			Name:     "Create permissive row policy for user using HTTP protocol on a cluster using replicated storage",
			ChEnv:    map[string]string{"CONFIGFILE": "config-replicated.xml"},
			Protocol: "http",
			Resource: resourcebuilder.New(resourceType, resourceName).
				WithStringAttribute("name", "test_policy").
				WithStringAttribute("database_name", "system").
				WithStringAttribute("table_name", "databases").
				WithStringAttribute("select_filter", "1").
				WithResourceFieldReference("grantee_user_name", "clickhousedbops_user", granteeUserName, "name").
				AddDependency(granteeUserResource.Build()).
				Build(),
			ResourceName:        resourceName,
			ResourceAddress:     fmt.Sprintf("%s.%s", resourceType, resourceName),
			CheckNotExistsFunc:  checkNotExistsFunc,
			CheckAttributesFunc: checkAttributesFunc,
		},
		{
			Name:     "Create restrictive row policy for user using HTTP protocol on a cluster using replicated storage",
			ChEnv:    map[string]string{"CONFIGFILE": "config-replicated.xml"},
			Protocol: "http",
			Resource: resourcebuilder.New(resourceType, resourceName).
				WithStringAttribute("name", "test_policy").
				WithStringAttribute("database_name", "system").
				WithStringAttribute("table_name", "databases").
				WithStringAttribute("select_filter", "name = 'default'").
				WithResourceFieldReference("grantee_user_name", "clickhousedbops_user", granteeUserName, "name").
				WithBoolAttribute("is_restrictive", true).
				AddDependency(granteeUserResource.Build()).
				Build(),
			ResourceName:        resourceName,
			ResourceAddress:     fmt.Sprintf("%s.%s", resourceType, resourceName),
			CheckNotExistsFunc:  checkNotExistsFunc,
			CheckAttributesFunc: checkAttributesFunc,
		},
		// Localfile storage, native
		{
			Name:        "Create permissive row policy for role using Native protocol on a cluster using localfile storage",
			ChEnv:       map[string]string{"CONFIGFILE": "config-localfile.xml"},
			ClusterName: &clusterName,
			Protocol:    "native",
			Resource: resourcebuilder.New(resourceType, resourceName).
				WithStringAttribute("cluster_name", clusterName).
				WithStringAttribute("name", "test_policy").
				WithStringAttribute("database_name", "system").
				WithStringAttribute("table_name", "databases").
				WithStringAttribute("select_filter", "name = 'default'").
				WithResourceFieldReference("grantee_role_name", "clickhousedbops_role", granteeRoleName, "name").
				AddDependency(granteeRoleResource.WithStringAttribute("cluster_name", clusterName).Build()).
				Build(),
			ResourceName:        resourceName,
			ResourceAddress:     fmt.Sprintf("%s.%s", resourceType, resourceName),
			CheckNotExistsFunc:  checkNotExistsFunc,
			CheckAttributesFunc: checkAttributesFunc,
		},
		{
			Name:        "Create permissive row policy for user using Native protocol on a cluster using localfile storage",
			ChEnv:       map[string]string{"CONFIGFILE": "config-localfile.xml"},
			ClusterName: &clusterName,
			Protocol:    "native",
			Resource: resourcebuilder.New(resourceType, resourceName).
				WithStringAttribute("cluster_name", clusterName).
				WithStringAttribute("name", "test_policy").
				WithStringAttribute("database_name", "system").
				WithStringAttribute("table_name", "tables").
				WithStringAttribute("select_filter", "database = 'system'").
				WithResourceFieldReference("grantee_user_name", "clickhousedbops_user", granteeUserName, "name").
				AddDependency(granteeUserResource.WithStringAttribute("cluster_name", clusterName).Build()).
				Build(),
			ResourceName:        resourceName,
			ResourceAddress:     fmt.Sprintf("%s.%s", resourceType, resourceName),
			CheckNotExistsFunc:  checkNotExistsFunc,
			CheckAttributesFunc: checkAttributesFunc,
		},
		{
			Name:        "Create restrictive row policy for user using Native protocol on a cluster using localfile storage",
			ChEnv:       map[string]string{"CONFIGFILE": "config-localfile.xml"},
			ClusterName: &clusterName,
			Protocol:    "native",
			Resource: resourcebuilder.New(resourceType, resourceName).
				WithStringAttribute("cluster_name", clusterName).
				WithStringAttribute("name", "test_policy").
				WithStringAttribute("database_name", "system").
				WithStringAttribute("table_name", "databases").
				WithStringAttribute("select_filter", "name != 'system'").
				WithResourceFieldReference("grantee_user_name", "clickhousedbops_user", granteeUserName, "name").
				WithBoolAttribute("is_restrictive", true).
				AddDependency(granteeUserResource.WithStringAttribute("cluster_name", clusterName).Build()).
				Build(),
			ResourceName:        resourceName,
			ResourceAddress:     fmt.Sprintf("%s.%s", resourceType, resourceName),
			CheckNotExistsFunc:  checkNotExistsFunc,
			CheckAttributesFunc: checkAttributesFunc,
		},
		// Localfile storage, http
		{
			Name:        "Create permissive row policy for role using HTTP protocol on a cluster using localfile storage",
			ChEnv:       map[string]string{"CONFIGFILE": "config-localfile.xml"},
			ClusterName: &clusterName,
			Protocol:    "http",
			Resource: resourcebuilder.New(resourceType, resourceName).
				WithStringAttribute("cluster_name", clusterName).
				WithStringAttribute("name", "test_policy").
				WithStringAttribute("database_name", "system").
				WithStringAttribute("table_name", "tables").
				WithStringAttribute("select_filter", "database = 'default'").
				WithResourceFieldReference("grantee_role_name", "clickhousedbops_role", granteeRoleName, "name").
				AddDependency(granteeRoleResource.WithStringAttribute("cluster_name", clusterName).Build()).
				Build(),
			ResourceName:        resourceName,
			ResourceAddress:     fmt.Sprintf("%s.%s", resourceType, resourceName),
			CheckNotExistsFunc:  checkNotExistsFunc,
			CheckAttributesFunc: checkAttributesFunc,
		},
		{
			Name:        "Create permissive row policy for user using HTTP protocol on a cluster using localfile storage",
			ChEnv:       map[string]string{"CONFIGFILE": "config-localfile.xml"},
			ClusterName: &clusterName,
			Protocol:    "http",
			Resource: resourcebuilder.New(resourceType, resourceName).
				WithStringAttribute("cluster_name", clusterName).
				WithStringAttribute("name", "test_policy").
				WithStringAttribute("database_name", "system").
				WithStringAttribute("table_name", "databases").
				WithStringAttribute("select_filter", "1").
				WithResourceFieldReference("grantee_user_name", "clickhousedbops_user", granteeUserName, "name").
				AddDependency(granteeUserResource.WithStringAttribute("cluster_name", clusterName).Build()).
				Build(),
			ResourceName:        resourceName,
			ResourceAddress:     fmt.Sprintf("%s.%s", resourceType, resourceName),
			CheckNotExistsFunc:  checkNotExistsFunc,
			CheckAttributesFunc: checkAttributesFunc,
		},
		{
			Name:        "Create restrictive row policy for user with grant option using HTTP protocol on a cluster using localfile storage",
			ChEnv:       map[string]string{"CONFIGFILE": "config-localfile.xml"},
			ClusterName: &clusterName,
			Protocol:    "http",
			Resource: resourcebuilder.New(resourceType, resourceName).
				WithStringAttribute("cluster_name", clusterName).
				WithStringAttribute("name", "test_policy").
				WithStringAttribute("database_name", "system").
				WithStringAttribute("table_name", "databases").
				WithStringAttribute("select_filter", "name = 'default'").
				WithResourceFieldReference("grantee_user_name", "clickhousedbops_user", granteeUserName, "name").
				WithBoolAttribute("is_restrictive", true).
				AddDependency(granteeUserResource.WithStringAttribute("cluster_name", clusterName).Build()).
				Build(),
			ResourceName:        resourceName,
			ResourceAddress:     fmt.Sprintf("%s.%s", resourceType, resourceName),
			CheckNotExistsFunc:  checkNotExistsFunc,
			CheckAttributesFunc: checkAttributesFunc,
		},
	}

	runner.RunTests(t, tests)
}
