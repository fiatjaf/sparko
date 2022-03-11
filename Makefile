dist: $(shell find . -name "*.go") spark-wallet/client/dist/app.js
	mkdir -p dist
	gox -ldflags="-s -w" -osarch="darwin/amd64 linux/386 linux/amd64 linux/arm linux/arm64 freebsd/amd64 openbsd/amd64" -output="dist/sparko_{{.OS}}_{{.Arch}}"

sparko: $(shell find . -name "*.go") spark-wallet/client/dist/app.js
	go build -o ./sparko

spark-wallet/client/dist/app.js: $(shell find spark-wallet/client/src)
	git submodule update
	cd spark-wallet/client/ && npm install
	cd spark-wallet/client && PATH=$$PATH:./node_modules/.bin/ ./build.sh

sparko-client-debug:
	git submodule update
	cd spark-wallet/client/ && npm install
	cd spark-wallet/client && NODE_ENV=development PATH=$$PATH:./node_modules/.bin/ ./build.sh
	go build -o ./sparko-client-debug
