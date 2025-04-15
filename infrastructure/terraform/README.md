# Scry API Database Infrastructure

This directory contains Terraform code for provisioning and configuring the PostgreSQL database infrastructure for the Scry API application on DigitalOcean.

## Prerequisites

- Terraform >= 1.0.0
- DigitalOcean API token with write access
- [Optional] Terraform state management solution (e.g., Terraform Cloud, DO Spaces)

## Setup

1. Copy `terraform.tfvars.example` to `terraform.tfvars` and fill in the required variables:
   ```
   cp terraform.tfvars.example terraform.tfvars
   ```

2. Edit `terraform.tfvars` with your DigitalOcean token and any customized settings.

3. Initialize Terraform:
   ```
   terraform init
   ```

4. Plan the deployment to see what will be created:
   ```
   terraform plan
   ```

5. Apply the configuration to create the infrastructure:
   ```
   terraform apply
   ```

6. After successful application, Terraform will output the database connection parameters:
   - Database host, port, name
   - Database user and password
   - Full connection string

## Safety Considerations

- **NEVER commit `terraform.tfvars` with secrets to version control**
- Consider using environment variables for sensitive values (set `TF_VAR_do_token`)
- For production deployments, use a secure backend for state storage

## Testing Migrations

To test migrations against the provisioned database:

1. Export the database connection string from Terraform outputs:
   ```
   export DATABASE_URL=$(terraform output -raw connection_string)
   ```

2. Run the migration test script:
   ```
   ../scripts/test-migrations.sh
   ```

## Common Operations

- To update the database configuration:
  ```
  terraform apply
  ```

- To destroy the database infrastructure (USE WITH CAUTION):
  ```
  terraform destroy
  ```
