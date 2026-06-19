#!/bin/bash
set -e

psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname postgres <<-EOSQL
    CREATE DATABASE tracker_db;
    GRANT ALL PRIVILEGES ON DATABASE tracker_db TO "$POSTGRES_USER";
EOSQL