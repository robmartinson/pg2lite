# pg2lite

pg2lite is a small command line utility to copy a pgsql database structure and data to a SQLite structure. Great for creating a small, fast snapshot of a larger pgsql database for local development, backups, or whatever.

At the moment, pg2lite only supports copying the structure and data of a pgsql database's public schema to a SQLite database It does not support copying indexes, constraints, or other database objects.

It does support tunneling through an SSH connection to the pgsql server, which is useful for copying databases from remote servers that can be accessed from a bastion host only.

## Build

`go build -o pg2lite cmd/pg2lite/main.go`

## Usage

All command line options can be viewed by running the command with the `-h` flag. The primary command is `migrate`, which runs the migration from pgsql to SQLite.

```
Usage:
  pg2lite migrate [flags]

Flags:
  -h, --help            help for migrate
      --sqlite string   SQLite output file (default "output.db")
      --with-data       Include data in migration

Global Flags:
      --db string         PostgreSQL database name
      --host string       PostgreSQL host (default "localhost")
      --output string     SQLite output file (default "output.db")
      --password string   PostgreSQL password
      --pg string         PostgreSQL connection string (optional)
      --port int          PostgreSQL port (default 5432)
      --sshhost string    SSH host
      --sshkey string     Path to SSH private key file
      --sshport int       SSH port (default 22)
      --sshuser string    SSH user
      --user string       PostgreSQL user

Use "pg2lite [command] --help" for more information about a command.
```

## Environment Variables

 Flags can be injected by using environment variables named as PG2LITE_ followed by the flag name in uppercase (e.g. `PG2LITE_DB` for `--db`). It will read environment variables from a local .env file if it exists.

Example .env file:

```
PG2LITE_USER = "user"
PG2LITE_PASSWORD = "password"
PG2LITE_HOST = "localhost"
PG2LITE_DB = "dbname"
PG2LITE_SSHHOST = "bastion-host"        # Optional
PG2LITE_SSHKEY = "/path/to/private/key" # Optional
PG2LITE_SSHUSER = "bastion-user"        # Optional
```

## Examples

Copy a local pgsql database to a SQLite database mirroring table structure only:

```
pg2lite migrate --host localhost --user user --password password --db mydb --output mydb.db
```

Copy a local pgsql database to a SQLite database mirroring table structure and data:

```
pg2lite migrate --host localhost --user user --password password --db mydb --output mydb.db --with-data
```

Copy a remote pgsql database to a SQLite database using an SSH tunnel:

```
pg2lite migrate --host remotehost --user user --password password --db mydb --output mydb.db --sshhost bastion-host --sshuser bastion-user --sshkey /path/to/private/key
```

Copy a remote pgsql database to a SQLite database using a full connection string:

```
pg2lite migrate --pg "host=remotehost user=user password=password dbname=mydb" --output mydb.db
```

