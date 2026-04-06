.PHONY: build test clean install smoke build-versioned

BIN := tf-summarize
PKG := .
VERSION ?= dev

build:
	go build -o $(BIN) $(PKG)

build-versioned:
	go build -ldflags "-X main.Version=$(VERSION)" -o $(BIN) $(PKG)

test:
	go test -v ./...

clean:
	rm -f $(BIN)

install:
	go install $(PKG)

smoke: build
	@echo "=== Plan: create ==="
	@echo 'Plan: 3 to add, 0 to change, 0 to destroy.' | \
		TF_WORKSPACE=dev TF_PHASE=plan ./$(BIN)
	@echo ""
	@echo "=== Plan: no changes ==="
	@echo 'No changes. Your infrastructure matches the configuration.' | \
		TF_WORKSPACE=dev TF_PHASE=plan ./$(BIN)
	@echo ""
	@echo "=== Apply: success ==="
	@printf 'module.s3.aws_s3_bucket.main: Creating...\nmodule.s3.aws_s3_bucket.main: Creation complete after 2s [id=bucket]\n\nApply complete! Resources: 1 added, 0 changed, 0 destroyed.\n' | \
		TF_WORKSPACE=prod TF_PHASE=apply ./$(BIN)
	@echo ""
	@echo "=== Apply: failure ==="
	@printf 'module.rds.aws_db_instance.main: Creating...\n\nError: creating RDS DB Instance: DBInstanceAlreadyExists\n\n  with module.rds.aws_db_instance.main,\n  on main.tf line 1\n' | \
		TF_WORKSPACE=prod TF_PHASE=apply ./$(BIN)
