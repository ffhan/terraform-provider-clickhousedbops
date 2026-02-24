terraform {
  required_providers {
    clickhousedbops = {
      source  = "ClickHouse/clickhousedbops"
      version = "~> 1.0"
    }
  }
}

provider "clickhousedbops" {
  protocol = var.clickhouse_protocol
  host     = var.clickhouse_host
  port     = var.clickhouse_port
  
  auth_config = {
    strategy = "password"
    username = var.clickhouse_username
    password = var.clickhouse_password
  }
}

# Create user 'a'
resource "clickhousedbops_user" "user_a" {
  name                 = "user_a"
  password_sha256_hash = var.user_a_password_hash
}

# Create user 'b'
resource "clickhousedbops_user" "user_b" {
  name                 = "user_b"
  password_sha256_hash = var.user_b_password_hash
}

# Create admin user (if not already exists)
resource "clickhousedbops_user" "user_admin" {
  name                 = "user_admin"
  password_sha256_hash = var.user_admin_password_hash
}

# Grant SELECT privilege to user 'a' on testdb.users
resource "clickhousedbops_grant_privilege" "user_a_select" {
  privilege_name    = "SELECT"
  database_name     = "testdb"
  table_name        = "users"
  grantee_user_name = clickhousedbops_user.user_a.name
}

# Grant SELECT privilege to user 'b' on testdb.users
resource "clickhousedbops_grant_privilege" "user_b_select" {
  privilege_name    = "SELECT"
  database_name     = "testdb"
  table_name        = "users"
  grantee_user_name = clickhousedbops_user.user_b.name
}

# Grant SELECT privilege to admin on testdb.users
resource "clickhousedbops_grant_privilege" "user_admin_select" {
  privilege_name    = "SELECT"
  database_name     = "testdb"
  table_name        = "users"
  grantee_user_name = clickhousedbops_user.user_admin.name
}

# Row policy for user 'a' - can only see their own rows
resource "clickhousedbops_row_policy" "user_a_policy" {
  name               = "user_a_row_policy"
  database_name      = "testdb"
  table_name         = "users"
  select_filter      = "user_id = 'a'"
  is_restrictive     = false
  grantee_user_names = [clickhousedbops_user.user_a.name]
}

# Row policy for user 'b' - can only see their own rows
resource "clickhousedbops_row_policy" "user_b_policy" {
  name               = "user_b_row_policy"
  database_name      = "testdb"
  table_name         = "users"
  select_filter      = "user_id = 'b'"
  is_restrictive     = false
  grantee_user_names = [clickhousedbops_user.user_b.name]
}

# Row policy for admin - permissive policy to see everything (USING 1)
resource "clickhousedbops_row_policy" "admin_full_access" {
  name               = "admin_full_access_policy"
  database_name      = "testdb"
  table_name         = "users"
  select_filter      = "1"
  is_restrictive     = false
  grantee_user_names = [clickhousedbops_user.user_admin.name]
}

output "user_a" {
  value = clickhousedbops_user.user_a.name
}

output "user_b" {
  value = clickhousedbops_user.user_b.name
}

output "user_admin" {
  value = clickhousedbops_user.user_admin.name
}

output "row_policies_created" {
  value = "user_a: ${clickhousedbops_row_policy.user_a_policy.name}, user_b: ${clickhousedbops_row_policy.user_b_policy.name}, admin: ${clickhousedbops_row_policy.admin_full_access.name}"
}
