# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
ENABLED_CGO=0
BINARY_NAME=ipchub
BINARY_DIR= bin/v1.0.0

build:
	CGO_ENABLED=$(ENABLED_CGO) $(GOBUILD) -o bin/$(BINARY_NAME) .
	cp -r demos bin/
	cp -r docs bin/
# linux compilation
build-linux-amd64:
	CGO_ENABLED=$(ENABLED_CGO) GOOS=linux GOARCH=amd64 $(GOBUILD) -o $(BINARY_DIR)/linux/amd64/$(BINARY_NAME)$(VERSION) .
	cp -r demos $(BINARY_DIR)/linux/amd64/
	cp -r docs $(BINARY_DIR)/linux/amd64/
build-linux-386:
	CGO_ENABLED=$(ENABLED_CGO) GOOS=linux GOARCH=386 $(GOBUILD) -o $(BINARY_DIR)/linux/386/$(BINARY_NAME)$(VERSION) .
build-linux-arm:
	CGO_ENABLED=$(ENABLED_CGO) GOOS=linux GOARCH=arm $(GOBUILD) -o $(BINARY_DIR)/linux/arm/$(BINARY_NAME)$(VERSION) .

# window compilation
build-windows-amd64:
	CGO_ENABLED=$(ENABLED_CGO) GOOS=windows GOARCH=amd64 $(GOBUILD) -o $(BINARY_DIR)/windows/amd64/$(BINARY_NAME)$(VERSION).exe .
	cp -r demos $(BINARY_DIR)/windows/amd64/
	cp -r docs $(BINARY_DIR)/windows/amd64/
build-windows-386:
	CGO_ENABLED=$(ENABLED_CGO) GOOS=windows GOARCH=386 $(GOBUILD) -o $(BINARY_DIR)/windows/386/$(BINARY_NAME)$(VERSION).exe .

# darwin compilation
build-darwin-amd64:
	CGO_ENABLED=$(ENABLED_CGO) GOOS=darwin GOARCH=amd64 $(GOBUILD) -o $(BINARY_DIR)/darwin/amd64/$(BINARY_NAME)$(VERSION) .
	cp -r demos $(BINARY_DIR)/darwin/amd64/
	cp -r docs $(BINARY_DIR)/darwin/amd64/
build-darwin-386:
	CGO_ENABLED=$(ENABLED_CGO) GOOS=darwin GOARCH=386 $(GOBUILD) -o $(BINARY_DIR)/darwin/386/$(BINARY_NAME)$(VERSION) .

# amd64 all platform compilation
build-amd64: build-linux-amd64 build-windows-amd64 build-darwin-amd64

# all 
build-all: build-linux-amd64 build-windows-amd64 build-darwin-amd64 build-linux-386 build-windows-386 build-darwin-386 build-linux-arm

test:
	$(GOTEST) -v ./...
clean:
	$(GOCLEAN)
	rm -f bin/$(BINARY_NAME)
	rm -rf $(BINARY_DIR)