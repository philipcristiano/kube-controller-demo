FROM golang:stretch AS build-env
ADD . /src
RUN cd /src && go build -o goapp

# final stage
FROM debian:stretch
WORKDIR /app
COPY --from=build-env /src/goapp /app/
ENTRYPOINT ./goapp
