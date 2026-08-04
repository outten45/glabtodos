BINARY := glabtodos
DIST_DIR := dist

.PHONY: all build build-linux build-darwin build-windows run clean

all: build

build: build-linux build-darwin build-windows

build-linux:
	@mkdir -p $(DIST_DIR)
	GOOS=linux GOARCH=amd64 go build -o $(DIST_DIR)/$(BINARY)-linux-amd64 .

build-darwin:
	@mkdir -p $(DIST_DIR)
	GOOS=darwin GOARCH=amd64 go build -o $(DIST_DIR)/$(BINARY)-darwin-amd64 .

build-windows:
	@mkdir -p $(DIST_DIR)
	GOOS=windows GOARCH=amd64 go build -o $(DIST_DIR)/$(BINARY)-windows-amd64.exe .

run:
	go run .

clean:
	rm -rf $(DIST_DIR)
