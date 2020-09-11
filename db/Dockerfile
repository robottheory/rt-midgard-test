FROM timescale/timescaledb:1.7.4-pg12

ENV POSTGRES_DB=midgard

ENV POSTGRES_USER=midgard
ENV POSTGRES_PASSWORD=password

COPY ddl.sql /docker-entrypoint-initdb.d/
