.PHONY: build-AuthorizerFunction build-ChildrenFunction build-AllowanceTypesFunction build-RecordsFunction

# make の export で環境変数を設定（シェル依存のインライン記法を使わない Windows/Linux 共通方式）
export GOOS := linux
export GOARCH := amd64
export CGO_ENABLED := 0

build-AuthorizerFunction:
	go build -o $(ARTIFACTS_DIR)/bootstrap ./src/functions/authorizer/

build-ChildrenFunction:
	go build -o $(ARTIFACTS_DIR)/bootstrap ./src/functions/children/

build-AllowanceTypesFunction:
	go build -o $(ARTIFACTS_DIR)/bootstrap ./src/functions/allowance-types/

build-RecordsFunction:
	go build -o $(ARTIFACTS_DIR)/bootstrap ./src/functions/records/
