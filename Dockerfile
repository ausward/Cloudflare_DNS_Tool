# Stage 1: Build the Go container
FROM golang:1.22.1-alpine AS builder

WORKDIR /app

COPY . .

RUN go build -o FlareAPI

# Stage 2: Create a new image and copy the built binary
FROM alpine:latest



WORKDIR /app

COPY --from=builder /app/FlareAPI .

# Add any additional dependencies or configurations here

# Set the entrypoint for the container
ENTRYPOINT ["./FlareAPI"]


# Stage 3: Push the new image to a container registry
# Replace <your-registry> with your actual container registry
# Replace <your-image-name> with the desired name for your image
# Replace <your-image-tag> with the desired tag for your image
# Uncomment the following lines to push the image
# RUN docker login <your-registry>
# RUN docker build -t <your-registry>/<your-image-name>:<your-image-tag> .
# RUN docker push <your-registry>/<your-image-name>:<your-image-tag>