# Цели для проверки ЛР3. На Windows — из Git Bash или WSL.
DB_TEST_URL ?= postgres://provider_routers_user:provider_routers_password@localhost:5433/provider_routers_test?sslmode=disable

.PHONY: up check test test-db smoke

up:            ## поднять PostgreSQL, Adminer, MinIO
	docker compose up -d
	docker compose ps

check:         ## доступны ли образы из compose (ловит приватный реестр до up)
	docker compose config --images | xargs -n1 docker manifest inspect > /dev/null && echo "образы доступны"

test:          ## unit-тесты без БД и MinIO
	go test ./...

test-db: up    ## + интеграционные тесты репозитория на реальной PostgreSQL
	docker compose exec -T postgres psql -U provider_routers_user -d provider_routers -tc "SELECT 1 FROM pg_database WHERE datname='provider_routers_test'" | grep -q 1 \
	  || docker compose exec -T postgres psql -U provider_routers_user -d provider_routers -c "CREATE DATABASE provider_routers_test"
	TEST_DATABASE_URL="$(DB_TEST_URL)" go test -count=1 ./internal/repository/

smoke:         ## полный сценарий против запущенного стека (docker compose up -d + go run .)
	bash scripts/smoke.sh
