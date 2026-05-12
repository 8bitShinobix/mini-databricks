include .env
export

# Change this depending on environment:
# Local dev  → http://localhost:8080
# Kind/K8s   → http://localhost:80
API_BASE ?= http://localhost:8080
PYTHON_JOBS_DIR ?= /Users/durgeshchandrakar/Documents/Coding/building_my_own_x/mini-databricks/sdk/python/jobs
TEST_EMAIL ?= test@test.com
TEST_PASSWORD ?= password123
WORKSPACE_NAME ?= my-workspace
WORKSPACE_PLAN ?= free
WORKSPACE_ID ?=
DATASET_NAME ?= sales-data
DATASET_FORMAT ?= csv
DATASET_ID ?=
JOB_ENTRYPOINT ?= $(PYTHON_JOBS_DIR)/analysis.py
JOB_PARAMETERS ?= {"region":"IN"}
JOB_COMPUTE ?= {"cpu":4,"memory_gb":16,"workers":3}
JOB_MAX_RETRIES ?= 3
JOB_ID ?=
ARTIFACT_ID ?=

.PHONY: docker-up docker-down migrate-up migrate-down migrate-create sqlc-generate \
        dev scheduler worker build-worker load-worker build-api load-api \
        build-scheduler load-scheduler build-migrate load-migrate build-all load-all \
        k8s-apply-secrets k8s-apply-infra k8s-apply-services k8s-apply-ingress \
        k8s-delete-secrets k8s-delete-infra k8s-delete-services \
        run-migrations kind-create kind-delete kind-setup \
        lint test ci metrics-api metrics-scheduler metrics-worker metrics-autoscaler metrics-all \
        register login me create-workspace list-workspaces \
        initiate-dataset list-datasets create-job list-jobs cancel-job job-progress \
        job-artifacts artifact-download smoke-api

# ── Docker ────────────────────────────────────────────────────────────────────
docker-up:
	docker compose -f deployments/docker/docker-compose.yml up -d

docker-down:
	docker compose -f deployments/docker/docker-compose.yml down

# ── Database ──────────────────────────────────────────────────────────────────
migrate-up:
	migrate -path internal/db/migrations -database "$(DB_URL)" up

migrate-down:
	migrate -path internal/db/migrations -database "$(DB_URL)" down 1

migrate-create:
	migrate create -ext sql -dir internal/db/migrations -seq $(name)

sqlc-generate:
	sqlc generate

# ── Local dev ─────────────────────────────────────────────────────────────────
dev:
	air

scheduler:
	go run ./services/scheduler/cmd/

worker:
	go run ./services/worker/cmd/

autoscaler:
	go run ./services/autoscaler/cmd/

cleanup:
	go run ./services/cleanup/cmd/

# ── Docker image builds ───────────────────────────────────────────────────────
build-api:
	docker build -f deployments/docker/Dockerfile.api -t mini-databricks-api:latest .

build-scheduler:
	docker build -f deployments/docker/Dockerfile.scheduler -t mini-databricks-scheduler:latest .

build-worker:
	docker build -f deployments/docker/Dockerfile.worker -t mini-databricks-worker:latest .

build-migrate:
	docker build -f deployments/docker/Dockerfile.migrate -t mini-databricks-migrate:latest .

build-all:
	make build-api
	make build-scheduler
	make build-worker
	make build-migrate

# ── Kind image loading ────────────────────────────────────────────────────────
load-api:
	kind load docker-image mini-databricks-api:latest --name mini-databricks

load-scheduler:
	kind load docker-image mini-databricks-scheduler:latest --name mini-databricks

load-worker:
	kind load docker-image mini-databricks-worker:latest --name mini-databricks

load-migrate:
	kind load docker-image mini-databricks-migrate:latest --name mini-databricks

load-all:
	make load-api
	make load-scheduler
	make load-worker
	make load-migrate

# ── Kubernetes apply ──────────────────────────────────────────────────────────
k8s-apply-secrets:
	kubectl apply -f deployments/k8s/secrets.yaml

k8s-apply-infra:
	kubectl apply -f deployments/k8s/infrastructure.yaml

k8s-apply-services:
	kubectl apply -f deployments/k8s/services.yaml

k8s-apply-ingress:
	kubectl apply -f deployments/k8s/ingress.yaml

# ── Kubernetes delete ─────────────────────────────────────────────────────────
k8s-delete-secrets:
	kubectl delete -f deployments/k8s/secrets.yaml

k8s-delete-infra:
	kubectl delete -f deployments/k8s/infrastructure.yaml

k8s-delete-services:
	kubectl delete -f deployments/k8s/services.yaml

# ── Kubernetes deploy ─────────────────────────────────────────────────────────
k8s-deploy:
	make load-all
	make k8s-apply-secrets
	make k8s-apply-infra
	make run-migrations
	make k8s-apply-services
	make k8s-apply-ingress

# ── Migrations ────────────────────────────────────────────────────────────────
run-migrations:
	kubectl delete job db-migrate --ignore-not-found
	kubectl apply -f deployments/k8s/migrate.yaml
	kubectl wait --for=condition=complete job/db-migrate --timeout=60s
	kubectl logs job/db-migrate

# ── Kind cluster ──────────────────────────────────────────────────────────────
kind-create:
	kind create cluster --name mini-databricks --config deployments/kind-config.yaml

kind-delete:
	kind delete cluster --name mini-databricks

kind-setup:
	make kind-create
	make build-all
	kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/main/deploy/static/provider/kind/deploy.yaml
	kubectl wait --namespace ingress-nginx --for=condition=ready pod --selector=app.kubernetes.io/component=controller --timeout=120s
	make k8s-deploy

# ── CI checks ─────────────────────────────────────────────────────────────────
lint:
	go vet ./...

test:
	go test ./... -v

ci:
	make lint
	make test
	make build-all
# ── Metrics ───────────────────────────────────────────────────────────────────
metrics-api:
	curl -s http://localhost:8080/metrics | grep -E "jobs_|tasks_|runs_|python_"

metrics-scheduler:
	curl -s http://localhost:9091/metrics | grep -E "runs_|tasks_created"

metrics-worker:
	curl -s http://localhost:9095/metrics | grep -E "tasks_processed|python_|task_duration"

metrics-autoscaler:
	@curl -s http://localhost:9093/metrics | grep -E "pending_|running_|worker_replicas" || echo "autoscaler not running"

metrics-all:
	@echo "=== API Gateway ==="
	@curl -s http://localhost:8080/metrics | grep -E "jobs_|tasks_|runs_|python_" || echo "api not running"
	@echo ""
	@echo "=== Scheduler ==="
	@curl -s http://localhost:9091/metrics | grep -E "runs_|tasks_created" || echo "scheduler not running"
	@echo ""
	@echo "=== Worker ==="
	@curl -s http://localhost:9095/metrics | grep -E "tasks_processed|python_|task_duration" || echo "worker not running"
	@echo ""
	@echo "=== Autoscaler ==="
	@curl -s http://localhost:9093/metrics | grep -E "pending_|running_|worker_replicas" || echo "autoscaler not running"
# ── API test helpers ──────────────────────────────────────────────────────────
register:
	curl -s -X POST $(API_BASE)/api/v1/auth/register \
		-H "Content-Type: application/json" \
		-d "{\"email\":\"$(TEST_EMAIL)\",\"password\":\"$(TEST_PASSWORD)\"}" \
		| python3 -m json.tool

login:
	@TOKEN=$$(curl -s -X POST $(API_BASE)/api/v1/auth/login \
		-H "Content-Type: application/json" \
		-d "{\"email\":\"$(TEST_EMAIL)\",\"password\":\"$(TEST_PASSWORD)\"}" \
		| grep -o '"token":"[^"]*"' | cut -d'"' -f4) && \
		echo $$TOKEN > .token && \
		echo "logged in, token saved to .token"

me:
	curl -s $(API_BASE)/api/v1/me \
		-H "Authorization: Bearer $$(cat .token)" \
		| python3 -m json.tool

create-workspace:
	curl -s -X POST $(API_BASE)/api/v1/workspaces \
		-H "Authorization: Bearer $$(cat .token)" \
		-H "Content-Type: application/json" \
		-d "{\"name\":\"$(WORKSPACE_NAME)\",\"plan\":\"$(WORKSPACE_PLAN)\"}" \
		| python3 -m json.tool

list-workspaces:
	curl -s $(API_BASE)/api/v1/workspaces \
		-H "Authorization: Bearer $$(cat .token)" \
		| python3 -m json.tool

smoke-api:
	-@$(MAKE) register
	@$(MAKE) login
	@$(MAKE) me
	@$(MAKE) create-workspace
	@$(MAKE) list-workspaces

create-job:
	@[ -n "$(WORKSPACE_ID)" ] || (echo "WORKSPACE_ID is required"; exit 1)
	@[ -n "$(DATASET_ID)" ] || (echo "DATASET_ID is required"; exit 1)
	curl -s -X POST $(API_BASE)/api/v1/jobs \
		-H "Authorization: Bearer $$(cat .token)" \
		-H "Content-Type: application/json" \
		-d '{"workspace_id":"$(WORKSPACE_ID)","dataset_id":"$(DATASET_ID)","entrypoint":"$(JOB_ENTRYPOINT)","parameters":$(JOB_PARAMETERS),"compute":$(JOB_COMPUTE),"max_retries":$(JOB_MAX_RETRIES),"idempotency_key":"$(shell uuidgen)"}' \
		| python3 -m json.tool

list-jobs:
	@[ -n "$(WORKSPACE_ID)" ] || (echo "WORKSPACE_ID is required"; exit 1)
	curl -s "$(API_BASE)/api/v1/jobs?workspace_id=$(WORKSPACE_ID)" \
		-H "Authorization: Bearer $$(cat .token)" \
		| python3 -m json.tool

cancel-job:
	@[ -n "$(JOB_ID)" ] || (echo "JOB_ID is required"; exit 1)
	curl -s -X POST $(API_BASE)/api/v1/jobs/$(JOB_ID)/cancel \
		-H "Authorization: Bearer $$(cat .token)" \
		| python3 -m json.tool

initiate-dataset:
	@[ -n "$(WORKSPACE_ID)" ] || (echo "WORKSPACE_ID is required"; exit 1)
	curl -s -X POST $(API_BASE)/api/v1/datasets/initiate \
		-H "Authorization: Bearer $$(cat .token)" \
		-H "Content-Type: application/json" \
		-d "{\"workspace_id\":\"$(WORKSPACE_ID)\",\"name\":\"$(DATASET_NAME)\",\"file_format\":\"$(DATASET_FORMAT)\"}" \
		| python3 -m json.tool

list-datasets:
	@[ -n "$(WORKSPACE_ID)" ] || (echo "WORKSPACE_ID is required"; exit 1)
	curl -s "$(API_BASE)/api/v1/datasets?workspace_id=$(WORKSPACE_ID)" \
		-H "Authorization: Bearer $$(cat .token)" \
		| python3 -m json.tool

job-progress:
	@[ -n "$(WORKSPACE_ID)" ] || (echo "WORKSPACE_ID is required"; exit 1)
	@[ -n "$(JOB_ID)" ] || (echo "JOB_ID is required"; exit 1)
	curl -s "$(API_BASE)/api/v1/jobs/$(JOB_ID)/progress?workspace_id=$(WORKSPACE_ID)" \
		-H "Authorization: Bearer $$(cat .token)" \
		| python3 -m json.tool

job-artifacts:
	@[ -n "$(WORKSPACE_ID)" ] || (echo "WORKSPACE_ID is required"; exit 1)
	@[ -n "$(JOB_ID)" ] || (echo "JOB_ID is required"; exit 1)
	curl -s "$(API_BASE)/api/v1/jobs/$(JOB_ID)/artifacts?workspace_id=$(WORKSPACE_ID)" \
		-H "Authorization: Bearer $$(cat .token)" \
		| python3 -m json.tool

artifact-download:
	@[ -n "$(JOB_ID)" ] || (echo "JOB_ID is required"; exit 1)
	@[ -n "$(ARTIFACT_ID)" ] || (echo "ARTIFACT_ID is required"; exit 1)
	curl -s "$(API_BASE)/api/v1/jobs/$(JOB_ID)/artifacts/$(ARTIFACT_ID)/download" \
		-H "Authorization: Bearer $$(cat .token)" \
		| python3 -m json.tool
