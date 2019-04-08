all: prod

prod: $(shell find . -name "*.go") node_modules/spark-wallet
	go-bindata -tags full -prefix node_modules/spark-wallet/dist/www -o bindata.go node_modules/spark-wallet/dist/www/...
	mkdir -p dist
	gox -tags="full" -osarch="darwin/amd64 linux/386 linux/amd64 linux/arm windows/386 windows/amd64" -output="dist/sparko_full_{{.OS}}_{{.Arch}}" -ldflags "-X main.Version=$$(jq -r '.version' node_modules/spark-wallet/package.json)"
	gox -osarch="darwin/amd64 linux/386 linux/amd64 linux/arm windows/386 windows/amd64" -output="dist/sparko_lean_{{.OS}}_{{.Arch}}" -ldflags "-X main.Version=$$(git log --pretty=format:'%H' | head -n 1)"

sparko: $(shell find . -name "*.go") bindata.go
	go build -tags full -o ./sparko -ldflags "-X main.Version=$$(jq -r '.version' node_modules/spark-wallet/package.json)"

bindata.go: node_modules/spark-wallet
	go-bindata -tags full -debug -prefix node_modules/spark-wallet/dist/www -o bindata.go node_modules/spark-wallet/dist/www/...

node_modules/spark-wallet:
	npm install spark-wallet
