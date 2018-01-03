gofmt -l .
go vet .
go test -ldflags -s -cover
