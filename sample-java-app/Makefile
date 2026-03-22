ALLURE_HUB_URL ?= http://localhost:8080
PROJECT_ID     ?= sample-java
PROJECT_NAME   ?= Sample Java App
BUILD_ID       ?= $(shell date +%Y%m%d-%H%M%S)
RESULTS_DIR    ?= target/allure-results
PASS_RATE      ?= 100

.PHONY: test upload run clean

## test  — delete existing results, compile and run all JUnit tests (writes allure-results/)
##         Pass PASS_RATE=<0-100> to simulate a target pass percentage, e.g. make test PASS_RATE=80
test:
	rm -rf $(RESULTS_DIR)
	mvn test -q -DpassRate=$(PASS_RATE)

## upload — zip allure-results and post to allure-hub, then generate report
upload:
	ALLURE_HUB_URL=$(ALLURE_HUB_URL) \
	PROJECT_ID=$(PROJECT_ID) \
	PROJECT_NAME='$(PROJECT_NAME)' \
	BUILD_ID=$(BUILD_ID) \
	RESULTS_DIR=$(RESULTS_DIR) \
	bash upload-results.sh

## run   — test + upload in one step
run: test upload

## clean — remove Maven build output
clean:
	mvn clean -q
