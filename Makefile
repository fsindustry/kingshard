ROOT:=$(shell pwd)
COMMIT_HASH=$(shell git rev-parse --short HEAD || echo "GitNotFound")
BUILD_DATE=$(shell date '+%Y-%m-%d %H:%M:%S')
GO_LDFLAGS="-X \"main.BuildVersion=${COMMIT_HASH}\" -X \"main.BuildDate=$(BUILD_DATE)\""


all: build

build: kingshard
goyacc:
	go build -o ${ROOT}/bin/goyacc ${ROOT}/goyacc
	${ROOT}/bin/goyacc -o ${ROOT}/pkg/sqlparser/sql.go ${ROOT}/pkg/sqlparser/sql.y
	gofmt -w ${ROOT}/pkg/sqlparser/sql.go
kingshard:
	go build -mod=vendor -ldflags ${GO_LDFLAGS} -o ./bin/kingshard ./cmd/kingshard
clean:
	@rm -rf bin
test:
	go test ./go/... -race

.PHONY: all goyacc
