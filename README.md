# pg2lite

pg2lite is a small command line utility to copy a pgsql database structure and data to a SQLite structure. Great for creating a small, fast snapshot of a larger pgsql database for local development, backups, or whatever.

At the moment, pg2lite only supports copying the structure and data of a pgsql database to a SQLite database. It does not support copying indexes, constraints, or other database objects.

It does support tunneling through an SSH connection to the pgsql server, which is useful for copying databases from remote servers that can be accessed from a bastion host only.

## Build

`go build -o pg2lite cmd/pg2lite/main.go`

## Usage

All command line options can be viewed by running the command with the `-h` flag. Additionally they can be injected by using environment variables named as PG2LITE_ followed by the flag name in uppercase.

`./pg2lite --host localhost --port 5432 --user postgres --password password --database mydb --output mydb.sqlite`
