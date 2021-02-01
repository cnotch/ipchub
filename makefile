# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
ENABLED_CGO=0
BINARY_NAME=ipchub
VERSION=1.1.0
BINARY_DIR=bin/v$(VERSION)

build:
	CGO_ENABLED=$(ENABLED_CGO) $(GOBUILD) -o bin/$(BINARY_NAME) -ldflags "-X github.com/cnotch/ipchub/config.Version=$(VERSION)" .
	cp -r demos bin/
	cp -r docs bin/

build-docker:
	CGO_ENABLED=$(ENABLED_CGO) GOOS=linux GOARCH=amd64 $(GOBUILD) -o bin/docker/$(BINARY_NAME) -ldflags "-X github.com/cnotch/ipchub/config.Version=$(VERSION)" .
	cp -r demos bin/docker/
	cp -r docs bin/docker/
	docker build -t ipchub .
	rm -rf bin/docker
	
build-linux-amd64:
	CGO_ENABLED=$(ENABLED_CGO) GOOS=linux GOARCH=amd64 $(GOBUILD) -o $(BINARY_DIR)/linux/amd64/$(BINARY_NAME) -ldflags "-X github.com/cnotch/ipchub/config.Version=$(VERSION)" .
	cp -r demos $(BINARY_DIR)/linux/amd64/
	cp -r docs $(BINARY_DIR)/linux/amd64/
build-linux-386:
	CGO_ENABLED=$(ENABLED_CGO) GOOS=linux GOARCH=386 $(GOBUILD) -o $(BINARY_DIR)/linux/386/$(BINARY_NAME) -ldflags "-X github.com/cnotch/ipchub/config.Version=$(VERSION)" .
	cp -r demos $(BINARY_DIR)/linux/386/
	cp -r docs $(BINARY_DIR)/linux/386/
build-linux-arm64:
	CGO_ENABLED=$(ENABLED_CGO) GOOS=linux GOARCH=arm64 $(GOBUILD) -o $(BINARY_DIR)/linux/arm/$(BINARY_NAME) -ldflags "-X github.com/cnotch/ipchub/config.Version=$(VERSION)" .
	cp -r demos $(BINARY_DIR)/linux/arm64/
	cp -r docs $(BINARY_DIR)/linux/arm64/
build-linux-arm:
	CGO_ENABLED=$(ENABLED_CGO) GOOS=linux GOARCH=arm $(GOBUILD) -o $(BINARY_DIR)/linux/arm/$(BINARY_NAME) -ldflags "-X github.com/cnotch/ipchub/config.Version=$(VERSION)" .
	cp -r demos $(BINARY_DIR)/linux/arm/
	cp -r docs $(BINARY_DIR)/linux/arm/

# window compilation
build-windows-amd64:
	CGO_ENABLED=$(ENABLED_CGO) GOOS=windows GOARCH=amd64 $(GOBUILD) -o $(BINARY_DIR)/windows/amd64/$(BINARY_NAME).exe -ldflags "-X github.com/cnotch/ipchub/config.Version=$(VERSION)" .
	cp -r demos $(BINARY_DIR)/windows/amd64/
	cp -r docs $(BINARY_DIR)/windows/amd64/

# darwin compilation
build-darwin-amd64:
	CGO_ENABLED=$(ENABLED_CGO) GOOS=darwin GOARCH=amd64 $(GOBUILD) -o $(BINARY_DIR)/darwin/amd64/$(BINARY_NAME) -ldflags "-X github.com/cnotch/ipchub/config.Version=$(VERSION)" .
	cp -r demos $(BINARY_DIR)/darwin/amd64/
	cp -r docs $(BINARY_DIR)/darwin/amd64/

# amd64 all platform compilation
build-amd64: build-linux-amd64 build-windows-amd64 build-darwin-amd64

# all 
build-all: build-linux-amd64 build-windows-amd64 build-darwin-amd64 build-linux-386 build-linux-arm64 build-linux-arm

test:
	$(GOTEST) -v ./...
clean:
	$(GOCLEAN)
	rm -f bin/$(BINARY_NAME)
	rm -rf bin/demos
	rm -rf bin/docs
	rm -rf $(BINARY_DIR)