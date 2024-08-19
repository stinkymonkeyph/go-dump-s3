
# Use the official Golang image as a build stage
FROM golang:1.23 AS builder

# Set the working directory inside the container
WORKDIR /app

# Copy the Go module files and download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy the source code
COPY . .

# Build the Go binary
RUN CGO_ENABLED=0 GOOS=linux go build -o /go/bin/backup-app

# Use a minimal image for the final stage
FROM alpine:latest

# Install necessary packages, including mysql-client and cron
RUN apk --no-cache add ca-certificates mysql-client bash curl cron

# Copy the Go binary from the builder stage
COPY --from=builder /go/bin/backup-app /usr/local/bin/backup-app

# Copy the script to run the backup
COPY run_backup.sh /usr/local/bin/run_backup.sh

# Make the script executable
RUN chmod +x /usr/local/bin/run_backup.sh

# Add the cron job
RUN echo "0 * * * * /usr/local/bin/run_backup.sh >> /var/log/cron.log 2>&1" > /etc/crontabs/root

# Start cron and the application
CMD ["sh", "-c", "crond && tail -f /var/log/cron.log"]

