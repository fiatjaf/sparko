all: prod

prod: $(shell find . -name "*.go") node_modules/spark-wallet
	go-bindata -tags full -prefix node_modules/spark-wallet/dist/www -o bindata.go node_modules/spark-wallet/dist/www/...
	go build -tags full -o ./sparko -ldflags "-X main.Version=$$(jq -r '.version' node_modules/spark-wallet/package.json)"

dist: $(shell find . -name "*.go") node_modules/spark-wallet
	go-bindata -tags full -prefix node_modules/spark-wallet/dist/www -o bindata.go node_modules/spark-wallet/dist/www/...
	mkdir -p dist
	gox -tags="full" -osarch="darwin/amd64 linux/386 linux/amd64 linux/arm" -output="dist/sparko_full_{{.OS}}_{{.Arch}}" -ldflags "-X main.Version=$$(jq -r '.version' node_modules/spark-wallet/package.json)"
	gox -osarch="darwin/amd64 linux/386 linux/amd64 linux/arm" -output="dist/sparko_lean_{{.OS}}_{{.Arch}}" -ldflags "-X main.Version=$$(git log --pretty=format:'%H' | head -n 1)"

sparko: $(shell find . -name "*.go") node_modules/spark-wallet
	go-bindata -tags full -debug -prefix node_modules/spark-wallet/dist/www -o bindata.go node_modules/spark-wallet/dist/www/...
	go build -tags full -o ./sparko -ldflags "-X main.Version=$$(jq -r '.version' node_modules/spark-wallet/package.json)"

node_modules/spark-wallet:
	npm install spark-wallet
