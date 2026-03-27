.PHONY: build-AuthorizerFunction build-ChildrenFunction build-AllowanceTypesFunction build-RecordsFunction

build-AuthorizerFunction:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o $(ARTIFACTS_DIR)/bootstrap ./src/functions/authorizer/

build-ChildrenFunction:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o $(ARTIFACTS_DIR)/bootstrap ./src/functions/children/

build-AllowanceTypesFunction:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o $(ARTIFACTS_DIR)/bootstrap ./src/functions/allowance-types/

build-RecordsFunction:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o $(ARTIFACTS_DIR)/bootstrap ./src/functions/records/
