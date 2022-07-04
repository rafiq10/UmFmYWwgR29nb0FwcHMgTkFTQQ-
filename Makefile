all: bin/url-collector
test: lint unit-test

PLATFORM=local

.PHONY: bin/url-collector
bin/url-collector:
	@docker build . --target bin \
	--output bin/ \
	--platform ${PLATFORM} 

.PHONY: unit-test
unit-test:
	@docker build . --target unit-test

.PHONY: unit-test-coverage
unit-test-coverage:
	@docker build . --target unit-test-coverage \
	--output coverage/
	cat coverage/cover.out
	--env-file ./envfile

.PHONY: lint
lint:
	@docker build . --target lint