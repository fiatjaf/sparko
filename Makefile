all: spark

prod: $(shell find . -name "*.go") node_modules/spark-wallet
	go-bindata -prefix node_modules/spark-wallet/dist/www -o bindata.go node_modules/spark-wallet/dist/www/...
	mkdir -p dist
	gox -os="linux darwin windows" -arch="386 amd64 arm" -output="dist/{{.Dir}}_{{.OS}}_{{.Arch}}"

spark: $(shell find . -name "*.go")
	go build -o ./spark -ldflags "-X main.Version=$$(jq -r '.version' node_modules/spark-wallet/package.json)"

bindata.go: node_modules/spark-wallet
	go-bindata -debug -prefix node_modules/spark-wallet/dist/www -o bindata.go node_modules/spark-wallet/dist/www/...

node_modules/spark-wallet:
	npm install spark-wallet
