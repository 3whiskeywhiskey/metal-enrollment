FROM nixos/nix:latest

# Install Go in the Nix environment
RUN nix-env -iA nixpkgs.go nixpkgs.git

WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=1 GOOS=linux go build -o builder ./cmd/builder

# The builder needs to run on NixOS with full Nix support
# So we use the nix image directly

WORKDIR /app
RUN mv /build/builder .

EXPOSE 8081

CMD ["./builder"]
