# Parameters
GOCMD=go
COPYCMD=cp
GO_VERSION=1.16.6
REAP=reap -vx
PASSH=passh
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOGET=$(GOCMD) get
COPY=$(COPYCMD)
BINARY_NAME=ansible-auth-wrapper
BINARY_DEST_SUB_DIR=bin
DEV_CMD=make test
BINARY_PATH=$(BINARY_DEST_SUB_DIR)/$(BINARY_NAME)
RUN_PORT=8383
DOMAINS=google.com


build: prep binary

prep:
	mkdir -p $(BINARY_DEST_SUB_DIR) || true

all: build

clean:
	$(GOCLEAN)
	rm -rf $(BINARY_DEST_SUB_DIR)/$(BINARY_NAME)


binary:
	$(GOBUILD) -o $(BINARY_DEST_SUB_DIR)/$(BINARY_NAME) -v

binary-cgo:
	CGO_ENABLED=1 $(GOBUILD) -o $(BINARY_DEST_SUB_DIR)/$(CGO_BINARY_NAME) -v

binary-no-cgo:
	CGO_ENABLED=0 $(GOBUILD) -o $(BINARY_DEST_SUB_DIR)/$(NO_CGO_BINARY_NAME) -v

run: build
	eval $(BINARY_PATH) 

help:
	eval $(BINARY_PATH) --help

test:	build help kill run
	
kill:
	pidof $(BINARY_NAME) && { killall $(BINARY_NAME); } || { true; } 

dev:
	./dev.sh 
