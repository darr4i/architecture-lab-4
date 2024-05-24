# ==== Build stage ====
FROM golang:1.22 as build

# Set the working directory
WORKDIR /go/src/practice-4

# Copy all files into the working directory
COPY . .

# Run tests
RUN go test ./...

# Set environment variable
ENV CGO_ENABLED=0

# Compile and install all binaries
RUN go install ./cmd/...

# ==== Final image ====
FROM alpine:latest

# Set the working directory
WORKDIR /opt/practice-4

# Copy the entry.sh script to the working directory in the final image
COPY entry.sh /opt/practice-4/

# Copy the compiled binaries from the build stage
COPY --from=build /go/bin/* /opt/practice-4

# Grant execute permissions to the entry.sh script
RUN chmod +x /opt/practice-4/entry.sh

# Run the command to check the contents of the directory
RUN ls /opt/practice-4

# Set the entry point for the container
ENTRYPOINT ["/opt/practice-4/entry.sh"]

# Set the default command
CMD ["server"]

