IMAGE           ?= ghcr.io/tlmanz/allure-hub
ALLURE_HUB_URL  ?= http://localhost:8080
TAG             ?= latest
VERSION         ?= $(TAG)
BUILD_TIME       = $(shell date -u +%Y-%m-%dT%H:%M:%SZ)

.PHONY: build frontend-build backend-build \
        test lint \
        docker-build docker-push \
        k8s-apply k8s-delete \
        dev-backend dev-frontend \
        sample-test sample-upload sample-run \
        clean

# ── Build ────────────────────────────────────────────────────────────────────

build: frontend-build backend-build

frontend-build:
	cd frontend && npm ci && npm run build

backend-build:
	$(MAKE) -C backend build

# ── Test / Lint ──────────────────────────────────────────────────────────────

test:
	$(MAKE) -C backend test

lint:
	$(MAKE) -C backend lint

# ── Docker ───────────────────────────────────────────────────────────────────
# Build context must be the repo root so the multi-stage Dockerfile can
# reach both frontend/ and backend/ directories.

docker-build:
	docker build \
	  --build-arg VERSION=$(VERSION) \
	  --build-arg BUILD_TIME=$(BUILD_TIME) \
	  -t $(IMAGE):$(TAG) .

docker-push: docker-build
	docker push $(IMAGE):$(TAG)

# ── Kubernetes ───────────────────────────────────────────────────────────────

k8s-apply:
	kubectl apply -f k8s/namespace.yaml
	kubectl apply -f k8s/pvc.yaml
	kubectl apply -f k8s/configmap.yaml
	kubectl apply -f k8s/deployment.yaml
	kubectl apply -f k8s/service.yaml
	kubectl apply -f k8s/ingress.yaml

k8s-delete:
	kubectl delete -f k8s/ --ignore-not-found

# ── Local dev ─────────────────────────────────────────────────────────────────

dev-backend:
	$(MAKE) -C backend run

dev-frontend:
	cd frontend && npm run dev

# ── Sample Java app ──────────────────────────────────────────────────────────

sample-test:
	$(MAKE) -C sample-java-app test

sample-upload:
	$(MAKE) -C sample-java-app upload ALLURE_HUB_URL=$(ALLURE_HUB_URL)

sample-run:
	$(MAKE) -C sample-java-app run ALLURE_HUB_URL=$(ALLURE_HUB_URL)

# ── Clean ─────────────────────────────────────────────────────────────────────

clean:
	$(MAKE) -C backend clean
	rm -rf frontend/dist frontend/node_modules
