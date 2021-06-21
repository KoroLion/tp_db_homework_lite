FROM golang
WORKDIR /go/src/app
COPY . .
RUN go build tp_db_homework/src/main

FROM ubuntu:20.04
COPY --from=0 /go/src/app/main ./
COPY --from=0 /go/src/app/init.sql ./

ARG DEBIAN_FRONTEND=noninteractive
RUN apt-get update && apt-get install -y postgresql postgresql-contrib

RUN echo "listen_addresses='localhost'" >> /etc/postgresql/12/main/postgresql.conf
RUN echo "max_connections = 32" >> /etc/postgresql/12/main/postgresql.conf
RUN echo "shared_buffers = 512MB" >> /etc/postgresql/12/main/postgresql.conf
RUN echo "effective_cache_size = 1536MB" >> /etc/postgresql/12/main/postgresql.conf
RUN echo "maintenance_work_mem = 128MB" >> /etc/postgresql/12/main/postgresql.conf
RUN echo "checkpoint_completion_target = 0.9" >> /etc/postgresql/12/main/postgresql.conf
RUN echo "wal_buffers = 16MB" >> /etc/postgresql/12/main/postgresql.conf
RUN echo "default_statistics_target = 100" >> /etc/postgresql/12/main/postgresql.conf
RUN echo "random_page_cost = 1.1" >> /etc/postgresql/12/main/postgresql.conf
RUN echo "effective_io_concurrency = 200" >> /etc/postgresql/12/main/postgresql.conf
RUN echo "work_mem = 4MB" >> /etc/postgresql/12/main/postgresql.conf
RUN echo "min_wal_size = 1GB" >> /etc/postgresql/12/main/postgresql.conf
RUN echo "max_wal_size = 4GB" >> /etc/postgresql/12/main/postgresql.conf
RUN echo "max_worker_processes = 8" >> /etc/postgresql/12/main/postgresql.conf
RUN echo "max_parallel_workers_per_gather = 4" >> /etc/postgresql/12/main/postgresql.conf
RUN echo "max_parallel_workers = 8" >> /etc/postgresql/12/main/postgresql.conf
RUN echo "max_parallel_maintenance_workers = 4" >> /etc/postgresql/12/main/postgresql.conf

USER postgres
RUN service postgresql start && psql -f init.sql

USER root
EXPOSE 5000
CMD service postgresql start && ./main
