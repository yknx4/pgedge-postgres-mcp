# Testing the Development MCP with Codex

This configuration runs the current checkout as a local stdio MCP server in
Codex, with strict environment-backed database credentials and PII masking.

## 1. Build the development binary

Run this again after changing the branch:

```bash
cd /Users/yknx4/src/yknx4/pgedge-postgres-mcp
ASDF_GOLANG_VERSION=1.26.4 go build \
  -o bin/pgedge-postgres-mcp-pii-dev \
  ./cmd/pgedge-pg-mcp-svr
```

## 2. Create the server configuration

Create `bin/pgedge-postgres-mcp-pii-dev.yaml`:

```yaml
databases:
  - name: pii_test
    host: "${PGEDGE_TEST_DB_HOST}"
    port: 5432
    database: "${PGEDGE_TEST_DB_NAME}"
    user: "${PGEDGE_TEST_DB_USER}"
    password: "${PGEDGE_TEST_DB_PASSWORD}"
    sslmode: require
    allow_writes: false

pii:
  enabled: true
  columns:
    email: [alternate_email]
    phone: [work_phone]

builtins:
  tools:
    query_database: true
    execute_explain: true
    llm_connection_selection: false

embedding:
  enabled: false

knowledgebase:
  enabled: false

llm:
  enabled: false
```

Environment interpolation applies to every YAML string value. Use
`${VARIABLE}` for clarity. Variables are expanded in memory after YAML parsing;
the configuration file is not rewritten. Startup fails if a referenced
variable is unset or empty. This strict behavior prevents accidental startup
with blank credentials.

Existing explicit `PGEDGE_*` configuration overrides retain their documented
priority over values loaded from YAML.

## 3. Register the MCP server in Codex

Add this to `~/.codex/config.toml`:

```toml
[mcp_servers.pgedge_postgres_pii_dev]
command = "/Users/yknx4/src/yknx4/pgedge-postgres-mcp/bin/pgedge-postgres-mcp-pii-dev"
args = [
  "-config",
  "/Users/yknx4/src/yknx4/pgedge-postgres-mcp/bin/pgedge-postgres-mcp-pii-dev.yaml",
]
cwd = "/Users/yknx4/src/yknx4/pgedge-postgres-mcp"
startup_timeout_sec = 30
tool_timeout_sec = 120
enabled = true

[mcp_servers.pgedge_postgres_pii_dev.env]
PGEDGE_TEST_DB_HOST = "localhost"
PGEDGE_TEST_DB_NAME = "your_database"
PGEDGE_TEST_DB_USER = "your_user"
PGEDGE_TEST_DB_PASSWORD = "replace-me"
```

For credentials that should not be stored in `config.toml`, export them in the
environment that launches Codex and forward their names instead:

```toml
[mcp_servers.pgedge_postgres_pii_dev]
env_vars = [
  "PGEDGE_TEST_DB_HOST",
  "PGEDGE_TEST_DB_NAME",
  "PGEDGE_TEST_DB_USER",
  "PGEDGE_TEST_DB_PASSWORD",
]
```

Do not configure both the inline `env` table and `env_vars` for the same
variable. Restart Codex after changing MCP configuration or rebuilding the
binary.

## 4. Verify PII masking

Run a SELECT through `query_database`:

```sql
SELECT email, first_name, last_name, phone_number
FROM users
LIMIT 5;
```

Recognized PII columns should contain realistic fake values. Detection uses
column names only. Additional column aliases can be added under `pii.columns`.
Repeated source values receive the same replacement within one response.

Set `pii.enabled` to `false`, restart the MCP server, and repeat the SELECT. The
original values should be returned with a warning that PII masking is disabled.

## 5. Verify performance-query bypass

Run the same query through `execute_explain`, then try `EXPLAIN` and `ANALYZE`.
Execution plans and statistics must remain untouched by PII processing.

## Supported built-in PII types

The default column-name categories are:

- `address`
- `city`
- `credit_card`
- `email`
- `first_name`
- `ip_address`
- `last_name`
- `name`
- `password`
- `phone`
- `postal_code`
- `ssn`
- `state`
- `token`
- `username`

The additional `generic` category retains the first and last character and
masks every character between them with `*`. Values shorter than three
characters are fully masked, while null values remain null.

Custom aliases use the same keys:

```yaml
pii:
  enabled: true
  columns:
    email: [billing_contact_email]
    phone: [emergency_contact_number]
    token: [oauth_token]
    password: [legacy_password_hash]
    generic: [private_identifier]
```
