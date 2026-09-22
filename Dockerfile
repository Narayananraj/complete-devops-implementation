# Containerize the go application that we have created
# This is the Dockerfile that we will use to build the image
# and run the container

# ---- Build stage ----
FROM golang:1.25 AS base

# Set the working directory inside the container
WORKDIR /app

# Copy go.mod AND go.sum for reproducible, verified dependency downloads
COPY go.mod go.sum ./

# Download all the dependencies
RUN go mod download

# Copy the source code to the working directory
COPY . .

# Build the application (static binary, no CGO)
RUN CGO_ENABLED=0 GOOS=linux go build -o main .

#######################################################
# Reduce the image size using multi-stage builds
# We will use a distroless image to run the application
FROM gcr.io/distroless/base-debian12:nonroot

# Set working directory in the final stage too — required before COPY
WORKDIR /app

# Copy the binary from the previous stage
COPY --from=base /app/main .

# Copy the static files from the previous stage
COPY --from=base /app/static ./static

# Expose the port on which the application will run
EXPOSE 8080

# Command to run the application
CMD ["./main"]