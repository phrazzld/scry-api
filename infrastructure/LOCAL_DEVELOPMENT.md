# Local Database Development Environment Setup

This document provides instructions for setting up a local PostgreSQL database for development that mirrors the production configuration.

## Option 1: Docker-based Setup (Recommended)

### Prerequisites
- Docker and Docker Compose installed on your development machine

### Setup

1. Navigate to the `infrastructure/local_dev` directory:
   ```bash
   cd infrastructure/local_dev
   ```

2. Start the database:
   ```bash
   docker-compose up -d
   ```

3. Verify connection:
   ```bash
   psql postgresql://scryapiuser:local_development_password@localhost:5432/scry
   ```

4. Configure your application to use this local database:

   - Set the following environment variable for development:
   ```
   SCRY_DATABASE_URL=postgres://scryapiuser:local_development_password@localhost:5432/scry?sslmode=disable
   ```

   - Or update your `config.yaml` file:
   ```yaml
   database:
     url: postgres://scryapiuser:local_development_password@localhost:5432/scry?sslmode=disable
   ```

5. Run migrations against your local database:
   ```bash
   go run cmd/server/main.go -migrate=up
   ```

6. Verify migration status:
   ```bash
   go run cmd/server/main.go -migrate=status
   ```

## Option 2: Native PostgreSQL Installation

If you prefer to install PostgreSQL directly on your system:

1. Install PostgreSQL 15:
   - **macOS**: `brew install postgresql@15`
   - **Ubuntu/Debian**:
     ```
     sudo apt install -y postgresql-15 postgresql-contrib-15
     sudo apt install -y postgresql-15-pgvector
     ```

2. Start PostgreSQL service:
   - **macOS**: `brew services start postgresql@15`
   - **Ubuntu/Debian**: `sudo systemctl start postgresql`

3. Create database and user:
```bash
sudo -u postgres psql -c "CREATE USER scryapiuser WITH PASSWORD 'local_development_password';"
sudo -u postgres psql -c "CREATE DATABASE scry;"
sudo -u postgres psql -c "GRANT ALL PRIVILEGES ON DATABASE scry TO scryapiuser;"
sudo -u postgres psql -d scry -c "CREATE EXTENSION IF NOT EXISTS vector;"
```

4. Configure database parameters (adjust paths as needed):
```bash
echo "shared_buffers = 128MB" | sudo tee -a /etc/postgresql/15/main/conf.d/custom.conf
echo "work_mem = 16MB" | sudo tee -a /etc/postgresql/15/main/conf.d/custom.conf
echo "max_connections = 50" | sudo tee -a /etc/postgresql/15/main/conf.d/custom.conf
sudo systemctl restart postgresql
```

5. Configure your application as described in Option 1, step 4.

## Resetting Your Local Database

To reset your local database to a clean state:

### Docker option:
```bash
cd infrastructure/local_dev
docker-compose down -v
docker-compose up -d
go run cmd/server/main.go -migrate=up
```

### Native installation:
```bash
sudo -u postgres psql -c "DROP DATABASE scry;"
sudo -u postgres psql -c "CREATE DATABASE scry;"
sudo -u postgres psql -c "GRANT ALL PRIVILEGES ON DATABASE scry TO scryapiuser;"
sudo -u postgres psql -d scry -c "CREATE EXTENSION IF NOT EXISTS vector;"
go run cmd/server/main.go -migrate=up
```
