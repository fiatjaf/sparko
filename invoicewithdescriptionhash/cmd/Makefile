dist: $(shell find .. -name "*.go")
	mkdir -p dist
	gox -ldflags="-s -w" -osarch="darwin/amd64 linux/386 linux/amd64 linux/arm linux/arm64 freebsd/amd64" -output="dist/invoicewithdescriptionhash_{{.OS}}_{{.Arch}}"

invoicewithdescriptionhash: $(shell find . -name "*.go")
	go build -o ./invoicewithdescriptionhash
