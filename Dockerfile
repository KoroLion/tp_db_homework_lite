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

USER postgres
RUN service postgresql start && psql -f init.sql

USER root
EXPOSE 5000
CMD service postgresql start && ./main
