.PHONY: build-all deploy-all test clean

# Сборка всех компонентов  
build-all: build-waiter build-backend build-frontend

build-waiter:
	cd waiter-service && docker build -t waiter:latest .

build-backend:
	cd backend && docker build -t billing-backend:latest .

build-frontend:
	cd frontend && docker build -t billing-frontend:latest .

# Деплой в Kubernetes
deploy-all: deploy-waiter deploy-backend

deploy-waiter:
	kubectl apply -f waiter-service/k8s/

deploy-backend:
	kubectl apply -f k8s/

# Запуск инфраструктуры
infra-up:
	docker-compose up -d postgres prometheus grafana

# Тестирование
test-waiter:
	./scripts/test-waiter.sh

# Генерация нагрузки
load-test:
	./scripts/load-test.sh

# Очистка
clean:
	docker-compose down -v
	kubectl delete -f waiter-service/k8s/ --ignore-not-found
	kubectl delete -f k8s/ --ignore-not-found
