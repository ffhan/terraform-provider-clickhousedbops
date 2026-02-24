# ClickHouse Row Policies E2E Test

This is a complete, repeatable end-to-end test for row policy support.
It is split into three phases:

1. **Phase 1**: Deploy admin user with full access
2. **Phase 2**: Add user A with their row policy
3. **Phase 3**: Add user B with their row policy

This tests that the provider correctly handles running `terraform apply` multiple times without any errors.

## What the Test Does

1. Starts a ClickHouse server in Docker
2. Creates a test table with 6 sample rows
3. Uses Terraform to add users and row policies (in three phases)
4. After each phase, verifies that each user can only see the rows they're supposed to see
5. Saves detailed logs so you can see exactly what happened

## Getting Started

**TL;DR**: Just run:

```bash
chmod +x test.sh
./test.sh
```

The script handles Docker, Terraform and ClickHouse setup. When it's done, check `results/phase*.md` to see what happened.

## Requirements

- Docker and Docker Compose installed (or Podman with podman-compose)
- Terraform 1.0 or newer
- The provider binary built locally

Before running the test, open `.terraformrc` and update this line to point to your local provider directory:

```hcl
"ClickHouse/clickhousedbops" = "/path/to/terraform-provider-clickhousedbops"
```

## Test Data

The test creates a `testdb.users` table with 6 rows:

| id | user_id | name              | email              |
|----|---------|-------------------|--------------------|
| 1  | a       | User A            | a@example.com      |
| 2  | a       | User A - Second   | a2@example.com     |
| 3  | b       | User B            | b@example.com      |
| 4  | b       | User B - Second   | b2@example.com     |
| 5  | admin   | Admin User        | admin@example.com  |
| 6  | public  | Public Data       | public@example.com |

## Row Policies

Three row policies are created in phased stages:

1. **admin_full_access_policy** - Applies to `user_admin`
   - Condition: `1` (always true, permissive)
   - Visibility: Can see all 6 rows

2. **user_a_row_policy** - Applies to `user_a`
   - Condition: `user_id = 'a'` (permissive)
   - Visibility: Can see 2 rows (id 1, 2)

3. **user_b_row_policy** - Applies to `user_b`
   - Condition: `user_id = 'b'` (permissive)
   - Visibility: Can see 2 rows (id 3, 4)

All policies are **permissive** (OR logic), meaning a row is visible if **any** policy permits it.

## Test Results

After running `./test.sh`, check the `results/` directory:

- `phase1.md` - Admin-only policy applied
  - Terraform plan shows 4 resources to add (admin policy + 3 grants)
  - Results show user_admin can see all 6 rows

- `phase2.md` - Admin + user_a policy applied
  - Terraform plan shows 1 resource to add (user_a policy only)
  - Results show user_a can see 2 rows (a's data)

- `phase3.md` - Admin + user_a + user_b policies applied
  - Terraform plan shows 1 resource to add (user_b policy only)
  - Results show user_b can see 2 rows (b's data)

- `terraform-logs/phase*/` - Terraform init/apply logs for each phase
  - `phase1_init.log` - Terraform initialization log
  - `phase1_apply.log` - Terraform apply log
  - `phase2_apply.log` - Terraform apply log (no init, state preserved)
  - `phase3_apply.log` - Terraform apply log (no init, state preserved)
