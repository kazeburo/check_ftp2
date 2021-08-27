VERSION=0.0.2
LDFLAGS=-ldflags "-w -s -X main.version=${VERSION} "

all: check_ftp2

.PHONY: check_ftp2

check_ftp2: main.go
	go build $(LDFLAGS) -o check_ftp2 main.go

linux: main.go
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o check_ftp2 main.go

check:
	go test ./...

fmt:
	go fmt ./...

tag:
	git tag v${VERSION}
	git push origin v${VERSION}
	git push origin main
