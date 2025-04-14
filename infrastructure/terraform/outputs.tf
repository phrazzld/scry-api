output "database_cluster_id" {
  description = "ID of the PostgreSQL cluster"
  value       = digitalocean_database_cluster.postgres.id
}

output "database_name" {
  description = "Name of the database"
  value       = digitalocean_database_db.app_database.name
}

output "database_host" {
  description = "Host for the database connection"
  value       = digitalocean_database_cluster.postgres.host
}

output "database_port" {
  description = "Port for the database connection"
  value       = digitalocean_database_cluster.postgres.port
}

output "database_user" {
  description = "Username for the database connection"
  value       = digitalocean_database_user.app_user.name
}

output "database_password" {
  description = "Password for the database connection"
  value       = digitalocean_database_user.app_user.password
  sensitive   = true
}

output "database_ssl_mode" {
  description = "SSL mode for the database connection"
  value       = "require"
}

output "connection_string" {
  description = "PostgreSQL connection string for application"
  value       = "postgres://${digitalocean_database_user.app_user.name}:${digitalocean_database_user.app_user.password}@${digitalocean_database_cluster.postgres.host}:${digitalocean_database_cluster.postgres.port}/${digitalocean_database_db.app_database.name}?sslmode=require"
  sensitive   = true
}
