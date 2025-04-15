variable "do_token" {
  description = "DigitalOcean API token"
  type        = string
  sensitive   = true
}

variable "database_name" {
  description = "Name of the primary database"
  type        = string
  default     = "scry"
}

variable "cluster_name" {
  description = "Name of the PostgreSQL cluster"
  type        = string
  default     = "scry-db-prd"  # Use 'scry-db-dev' for dev environments
}

variable "pg_version" {
  description = "PostgreSQL version"
  type        = string
  default     = "15"  # Use latest stable version
}

variable "region" {
  description = "DigitalOcean region"
  type        = string
  default     = "nyc1"  # Choose a region close to your application
}

variable "node_size" {
  description = "Database node size"
  type        = string
  default     = "db-s-1vcpu-1gb"  # Smallest DB size for MVP - adjust based on needs
}

variable "node_count" {
  description = "Number of database nodes (use 1 for single node, 2+ for HA)"
  type        = number
  default     = 1  # Single node for MVP - adjust based on needs
}

variable "authorized_ips" {
  description = "List of IP addresses or ranges that can access the database"
  type        = list(string)
  default     = []  # Empty list means no firewall rules (unrestricted)
}

variable "backup_hour" {
  description = "Hour of day (UTC) to take automatic backup (0-23)"
  type        = number
  default     = 3  # 3 AM UTC
}

variable "backup_minute" {
  description = "Minute of hour to take automatic backup (0-59)"
  type        = number
  default     = 0  # On the hour
}

variable "alert_emails" {
  description = "Email addresses for monitoring alerts"
  type        = list(string)
  default     = ["admin@example.com"]  # Replace with actual emails
}

variable "database_password" {
  description = "Password for the application database user"
  type        = string
  sensitive   = true
}
