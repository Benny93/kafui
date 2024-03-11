FROM golang:1.22 as BuildStage

# Set destination for COPY
WORKDIR /app

# Download Go modules
COPY go.mod go.sum ./
RUN go mod download
COPY . ./

# Build
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags "-w -s" -o /kafui ./cmd/kafui

FROM scratch

WORKDIR /

COPY --from=BuildStage /kaf /bin/kafui

USER 1001

# Run
CMD ["/bin/kafui"]