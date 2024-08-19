
# Use the official Golang image with version 1.23 as a build stage
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

# Install necessary packages, including mysql-client
RUN apk --no-cache add ca-certificates mysql-client

# Copy the Go binary from the builder stage
COPY --from=builder /go/bin/backup-app /usr/local/bin/backup-app

# Set environment variables for your app configuration
ENV AWS_REGION=your_aws_region
ENV DISCORD_URL=your_discord_webhook_url
ENV S3_BUCKET=your_s3_bucket_name
ENV S3_PREFIX=your/s3/prefix
ENV MYSQL_USER=your_mysql_user
ENV MYSQL_PASSWORD=your_mysql_password
ENV MYSQL_HOST=your_mysql_host
ENV MYSQL_PORT=3306
ENV DATABASES=db1,db2,db3

# Run the binary
ENTRYPOINT ["/usr/local/bin/backup-app"]

