##################
# Build Stage 1/2   -> Dependencies
##################
FROM golang:1.19 AS dependencies

# Create appuser.
ENV USER=appuser
ENV UID=10001

RUN adduser \
    --disabled-password \
    --gecos "" \
    --home "/nonexistent" \
    --shell "/sbin/nologin" \
    --no-create-home \
    --uid "${UID}" \
    "${USER}"

WORKDIR /opt/app

COPY go.mod .
COPY go.sum .

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o /go/bin/app

RUN curl -L -s --output - https://github.com/jwilder/dockerize/releases/download/v0.6.1/dockerize-linux-amd64-v0.6.1.tar.gz | tar -C /usr/bin/ -xz

##################
# Build Stage 2/2   -> Production image
##################
FROM alpine:3.14 AS production

# Import the user and group files from the builder.
COPY --from=dependencies /etc/passwd /etc/passwd
COPY --from=dependencies /etc/group /etc/group

COPY --from=dependencies /go/bin/app /app

COPY --from=dependencies /usr/bin/dockerize /dockerize

# Use an unprivileged user.
USER appuser:appuser
