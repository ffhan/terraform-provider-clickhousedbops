variable "clickhouse_protocol" {
  type        = string
  description = "ClickHouse protocol (native or http)"
  default     = "native"
}

variable "clickhouse_host" {
  type        = string
  description = "ClickHouse server host"
  default     = "localhost"
}

variable "clickhouse_port" {
  type        = number
  description = "ClickHouse server port"
  default     = 9000
}

variable "clickhouse_username" {
  type        = string
  description = "ClickHouse username (should be default user with full privileges enabled)"
  default     = "default"
}

variable "clickhouse_password" {
  type        = string
  description = "ClickHouse password for default user"
  default     = "password"
}

variable "user_a_password_hash" {
  type        = string
  description = "SHA256 hash of password for user_a"
  default     = "5e884898da28047151d0e56f8dc629302540667b49fdf41121f6c3ee77c80612" # password: "password"
}

variable "user_b_password_hash" {
  type        = string
  description = "SHA256 hash of password for user_b"
  default     = "5e884898da28047151d0e56f8dc629302540667b49fdf41121f6c3ee77c80612" # password: "password"
}

variable "user_admin_password_hash" {
  type        = string
  description = "SHA256 hash of password for user_admin"
  default     = "5e884898da28047151d0e56f8dc629302540667b49fdf41121f6c3ee77c80612" # password: "password"
}
