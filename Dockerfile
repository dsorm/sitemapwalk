# use ubuntu focal as base image
# builder stage
FROM golang:1.20 AS builder

# make sure we're root
USER root

WORKDIR /usr/src/app
# copy source files
COPY . .

# get dependencies and compile
RUN go build -v -o /usr/local/bin/app github.com/dsorm/sitemapwalk
#chmod +x /home/root/go/bin/sitemapwalk

# final image stage
FROM ubuntu:jammy

# copy artefacts and needed files
RUN mkdir /app && mkdir /app/html
COPY --from=builder /usr/local/bin/app /app/sitemapwalk

# open port
EXPOSE 80

# register all args
ENV PGHOST=postgres PGDATABASE=sitemapwalk PGUSER=sitemapwalk PGPASSWORD=sitemapwalk

# run
WORKDIR /app
CMD ["./sitemapwalk", "run"]