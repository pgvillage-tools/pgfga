# PGFGA - Postgres Fine Grained Access tool

## License

PGFGA is licensed under the [GNU General Public License v3.0](LICENSE).

# Introduction

PGFGA is a tool to configure and manage Postgres logical objects (Users, Roles, Databases, Extensions and Replication Slots).
Users and roles can be synced from a ldap directory.

## The origin
Usually organizations manage their users in a directory (like AD, ldap, or a cloud IAM).
Although Postgres can integrate with such tools for authentication, it does not for authorization.
Furthermore, the users need to either be defined in postgres or be mapped to an existing user before authentication can finish successfully.

As such many enterprises have looked into syncing authorization into postgres and many tools exist.
At [bol.com](https://www.bol.com/) an existing tool based on python 2 was used.
For conformity, the tool was in-house rebuilt into python 3 and after that enhanced with unittests, and extra capabilities.

For the bol.com use case, it made sense to manage more types of objects next to users and roles.
As an example, managing ownership on a database means that the database should exist too.
The tool was expanded to manage databases, extensions and replication slots too.
And thus, pgcdfga was born, built, and maintained.

After a few years, Mannem Solutions has been offered the ability to adopt the solution.
We have renamed it to pgfga (PostGres Fine Grained Access), rebuilt it in GoLang and are now maintaining (and using) the new tool in our own solutions.

# Requirements

## PGFGA config

PGFGA can be configured using a configuration yaml.
In container environments (like Kubernetes), that could be a configmap mounted as a volume.
For more details on the configuration format, please refer to [our config description](CONFIG.md).

### Postgres User account

The PGFGA tool requires a postgres user with access and SUPERUSER privileges to run.
This user can be set up in one of these ways:
- Local access (when running inside of master container) and using ident authentication
- A staged username / password e.a. configured with setup.sql script (by setting PG_PRIMARY_USER, PG_PRIMARY_PASSWORD, etc.).
- A staged pgfga user with client certificate authentication.
- Setting up ldap authentication for the postgres user (see the [docker-compose example](docker-compose.yml) for an example of this setup).

# Downloading
The most straight forward option is to download the [pgfga](https://github.com/pgvillage-tools/pgfga) binary directly from the [github release page](https://github.com/pgvillage-tools/pgfga/releases).
But there are other options, like
- using the [container image from dockerhub](https://hub.docker.com/repository/docker/pgvillage-tools/pgfga/general)
- direct build from source (if you feel you must)

Please refer to [our download instructions](DOWNLOAD_AND_RUN.md) for more details on all options.

# Using
**Note** please refer to our [configuration documentation](CONFIG.md) to learn about all capabilities and configuration features of pgfga.

After downloading the binary to a folder in your path, you can run pgfga with a command like:
```bash
pgfga -c ./myconfig.yml
```

# Contributing
Please see [Developing](DEVELOP.md) for more information.
