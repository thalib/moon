## Overview
- Implement configuration architecture using Viper for single source of truth
- Support static config files (YAML/TOML) and environment variables for secrets
- Parse configuration into immutable, read-only AppConfig struct at startup

## Requirements
- Create `internal/config/config.go` with AppConfig struct
- Support `config.yaml` or `config.toml` for static configuration
- Support `.env` file and environment variables for secrets (DB credentials, JWT secrets)
- Configuration fields: server port, database connection string, JWT secret, API key settings
- Make AppConfig immutable after initialization (no runtime mutations)
- Implement thread-safe access to configuration
- Validate required configuration fields on startup
- Fail fast with clear error messages for missing required config

## Acceptance
- Configuration loads from YAML/TOML file successfully
- Environment variables override file-based config
- Missing required config causes startup failure with descriptive error
- AppConfig is read-only after initialization
- Unit tests cover config loading, validation, and error scenarios
