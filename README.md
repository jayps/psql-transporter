# psql-transporter

A small, friendly CLI to **export** a PostgreSQL database to SQL and **import** it into another database — with guard rails, prompts, and progress spinners.

- Works by shelling out to `pg_dump` and `psql` (reliable, version-compatible).
- Reads connection sources from a root-level YAML config.
- Blocks importing into **protected** destinations.
- Shows clear, step-by-step progress.

---

## Table of contents

- [Features](#features)
- [Requirements](#requirements)
- [Install](#install)
- [Configuration](#configuration)
- [Usage](#usage)
- [Examples](#examples)
- [How it works](#how-it-works)
- [Project layout](#project-layout)
- [Troubleshooting](#troubleshooting)
- [Development](#development)
- [Roadmap / ideas](#roadmap--ideas)
- [License](#license)

---

## Features

- ✅ Creates a default config on first run (`psql-transporter.yaml`)
- ✅ Interactive **source** and **destination** selection
- ✅ **Destructive-action warning** before wiping destination schema
- ✅ **Protected** flag on a source to prevent using it as destination
- ✅ Simple **spinners** and a progress message flow
- ✅ Minimal dependencies, idiomatic Go layout

---

## Requirements
- PostgreSQL client tools on your `PATH`:
  - `pg_dump`
  - `psql`

Install tips:
- macOS (Homebrew):  
  ```bash
  brew install libpq && brew link --force libpq
  ```
- Ubuntu/Debian:  
  ```bash
  sudo apt-get install postgresql-client
  ```
---

## Install
You can install this tool with homebrew:
```bash
brew tap jayps/psql-transporter https://github.com/jayps/homebrew-psql-transporter
brew install jayps/psql-transporter/psql-transporter
```

```bash
# clone your repo
git clone https://github.com/jayps/psql-transporter.git
cd psql-transporter

# initialize and get deps
go mod tidy

# build
go build -o bin/psql-transporter ./cmd/psql-transporter

# (or run directly)
go run ./cmd/psql-transporter
```

---

## Configuration

On first run, the tool looks for `psql-transporter.yaml` at the repo root.  
If missing, it writes a minimal default and exits so you can edit it.

### File: `psql-transporter.yaml`

```yaml
# psql-transporter.yaml
sources:
  - name: staging
    host: "staging.db.local"
    port: 5432
    user: "postgres"
    password: "postgres"
    dbname: "app_db"
    sslmode: "disable"   # disable | require | verify-ca | verify-full
    protected: true      # prevents being chosen as a DESTINATION

  - name: dev
    host: "127.0.0.1"
    port: 5432
    user: "postgres"
    password: "postgres"
    dbname: "app_db"
    sslmode: "disable"
    protected: false
```
---

## Usage

```bash
# run
go run ./cmd/psql-transporter
# or after building
./bin/psql-transporter
```

The flow:

1. Ensures `psql-transporter.yaml` exists (creates a default if not).
2. Loads sources, shows a prompt to pick **SOURCE**.
3. Shows a prompt to pick **DESTINATION**.
4. If destination is `protected: true`, it aborts.
5. Asks for **confirmation**: destination will be **wiped**.
6. Runs three steps with spinners:
   - Export source (`pg_dump`) → `./dump.sql`
   - Wipe destination schema (`DROP SCHEMA public CASCADE; CREATE SCHEMA public;`)
   - Import into destination (`psql -f dump.sql`)
7. Prints **All done ✅** if everything succeeds.

---

## Examples

Run and create a default config (first run):

```bash
go run ./cmd/psql-transporter
# → "Created default config at psql-transporter.yaml"
# Edit it, then run again
```

Normal run:

```bash
./bin/psql-transporter
# Choose SOURCE: staging
# Choose DESTINATION: dev
# Confirm wipe: "DESTINATION 'dev' will be WIPED ..." → yes
# Watch spinners for Exporting → Wiping → Importing
```

---

## How it works

- **Export**:  
  `pg_dump -h <host> -p <port> -U <user> -d <dbname> --no-owner --no-privileges -F p -f dump.sql`
- **Wipe destination**:  
  `psql -h <host> -p <port> -U <user> -d <dbname> -c "DROP SCHEMA public CASCADE; CREATE SCHEMA public;"`
- **Import**:  
  `psql -h <host> -p <port> -U <user> -d <dbname> -f dump.sql`
- **Auth & SSL**:
  - Sets `PGPASSWORD` and `PGSSLMODE` for each command.

---

**Key deps:**
- CLI: `github.com/spf13/cobra`
- Prompts: `github.com/AlecAivazis/survey/v2`
- YAML: `gopkg.in/yaml.v3`
- Spinners/pretty: `github.com/pterm/pterm`

---

## Troubleshooting

**`pg_dump` or `psql` not found.**  
Make sure PostgreSQL client tools are installed and on your `PATH`. See [Requirements](#requirements).

**Auth/SSL issues.**  
Double-check `host`, `port`, `user`, `password`, `dbname`, and `sslmode` in your YAML. Verify network access from your environment to the DB host.

**Extensions / roles / permissions.**  
`--no-owner --no-privileges` avoids permission errors in many cases, but if you rely on roles or extensions, you may need to:
- Pre-create extensions on destination, or
- Adjust dump flags to include what you need.

---

## Development

Common tasks (Makefile optional):

```makefile
build:
	go build -o bin/psql-transporter ./cmd/psql-transporter

run:
	go run ./cmd/psql-transporter

fmt:
	go fmt ./...

lint:
	go vet ./...
```

Module init & deps:

```bash
go mod init github.com/jayps/psql-transporter
go get github.com/spf13/cobra@latest
go get github.com/AlecAivazis/survey/v2@v2.3.7
go get gopkg.in/yaml.v3@v3.0.1
go get github.com/pterm/pterm@latest
```

---

## Roadmap / ideas

- `psql-transporter init` to scaffold a commented config file
- `--config path/to/file.yaml` flag
- `.env` support (e.g., interpolate `${ENV_VAR}` in YAML)
- `--dry-run` to print planned actions without running them
- Filter by `--schema` or `--table` (pass-through to `pg_dump`)
- Logging to file with `log/slog`
- Progress by bytes (parse `pg_dump`/`psql` output or monitor file growth)
- Optional compression of dumps

---

## License

MIT. See LICENSE.
