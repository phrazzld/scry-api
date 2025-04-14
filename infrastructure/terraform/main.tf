provider "digitalocean" {
  token = var.do_token
}

# Create PostgreSQL cluster on DigitalOcean
resource "digitalocean_database_cluster" "postgres" {
  name       = var.cluster_name
  engine     = "pg"
  version    = var.pg_version
  size       = var.node_size
  region     = var.region
  node_count = var.node_count

  # Database settings - adjust based on chosen node size
  # These are the additional settings available in DO managed PostgreSQL
  maintenance_window {
    day  = "Sunday"
    hour = "02:00:00"
  }

  backup_restore {
    backup_hour   = var.backup_hour
    backup_minute = var.backup_minute
  }
}

# Create application database
resource "digitalocean_database_db" "app_database" {
  cluster_id = digitalocean_database_cluster.postgres.id
  name       = var.database_name
}

# Create a user with appropriate permissions
resource "digitalocean_database_user" "app_user" {
  cluster_id = digitalocean_database_cluster.postgres.id
  name       = "scryapiuser"  # Specific app user with restricted permissions
}

# Firewall rules for database access
resource "digitalocean_database_firewall" "postgres_fw" {
  count      = length(var.authorized_ips) > 0 ? 1 : 0
  cluster_id = digitalocean_database_cluster.postgres.id

  dynamic "rule" {
    for_each = var.authorized_ips
    content {
      type  = "ip_addr"
      value = rule.value
    }
  }
}

# Connect to the database to enable pgvector extension
provider "postgresql" {
  host            = digitalocean_database_cluster.postgres.host
  port            = digitalocean_database_cluster.postgres.port
  database        = var.database_name
  username        = digitalocean_database_cluster.postgres.user
  password        = digitalocean_database_cluster.postgres.password
  sslmode         = "require"
  connect_timeout = 15
  superuser       = false

  # This depends_on ensures the database is created before attempting to connect
  depends_on = [
    digitalocean_database_db.app_database
  ]
}

# Enable pgvector extension
resource "postgresql_extension" "pgvector" {
  name     = "vector"
  database = var.database_name

  depends_on = [
    digitalocean_database_db.app_database
  ]
}

# Set up monitoring alerts
resource "digitalocean_monitor_alert" "cpu_alert" {
  alerts {
    email = var.alert_emails
  }
  window      = "5m"
  type        = "v1/insights/droplet/cpu"
  value       = 90
  compare     = "GreaterThan"
  description = "CPU usage is above 90%"
  entities    = [digitalocean_database_cluster.postgres.id]
}

resource "digitalocean_monitor_alert" "disk_alert" {
  alerts {
    email = var.alert_emails
  }
  window      = "5m"
  type        = "v1/insights/droplet/disk_utilization"
  value       = 90
  compare     = "GreaterThan"
  description = "Disk usage is above 90%"
  entities    = [digitalocean_database_cluster.postgres.id]
}

resource "digitalocean_monitor_alert" "memory_alert" {
  alerts {
    email = var.alert_emails
  }
  window      = "5m"
  type        = "v1/insights/droplet/memory_utilization"
  value       = 90
  compare     = "GreaterThan"
  description = "Memory usage is above 90%"
  entities    = [digitalocean_database_cluster.postgres.id]
}
