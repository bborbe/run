
default: precommit

precommit: ensure format generate test check addlicense
	@echo "ready to commit"

ensure:
	GO111MODULE=on go mod tidy

format:
	go install -mod=vendor github.com/incu6us/goimports-reviser
	find . -type f -name '*.go' -not -path './vendor/*' -exec gofmt -w "{}" +
	find . -type f -name '*.go' -not -path './vendor/*' -exec goimports-reviser -project-name bitbucket.apps.seibert-media.net -file-path "{}" \;

generate:
	rm -rf mocks avro
	go generate -mod=vendor ./...

test:
	go test -cover -race $(shell go list ./... | grep -v /vendor/)

check: lint vet errcheck

lint:
	go install -mod=vendor golang.org/x/lint/golint
	@GOFLAGS=-mod=vendor golint -min_confidence 1 $(shell go list ./... | grep -v /vendor/)

vet:
	go vet $(shell go list ./... | grep -v /vendor/)

errcheck:
	go install -mod=vendor github.com/kisielk/errcheck
	@GOFLAGS=-mod=vendor errcheck -ignore '(Close|Write|Fprint)' $(shell go list ./... | grep -v /vendor/)

addlicense:
	go install -mod=vendor  github.com/google/addlicense
	addlicense -c "Benjamin Borbe" -y 2021 -l bsd ./*.go
