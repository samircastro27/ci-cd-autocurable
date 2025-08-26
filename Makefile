.PHONY: build push deploy operator

build:
	docker build -t cuembybot/demo-microservice:latest ./microservice

push:
	docker push cuembybot/demo-microservice:latest

deploy:
	kubectl apply -f microservice/k8s

tekton-rbac:
	kubectl apply -f tekton/rbac.yaml

tekton: tekton-rbac
	kubectl apply -f tekton/pipeline.yaml
	kubectl apply -f tekton/triggers.yaml

operator:
	kubectl apply -f operator/config/crd.yaml
	kubectl apply -f operator/config/samples/healingpolicy.yaml

monitoring:
	kubectl apply -f monitoring/prometheus.yaml
	kubectl apply -f monitoring/grafana.yaml

all: build push deploy tekton operator monitoring

SAMPLE_PAYLOAD=tekton/sample_push_event.json
NGROK_PORT=31234
NGROK_URL_FILE=tekton/.ngrok_url
NGROK_AUTH_TOKEN?=  # (opcional) si usas cuenta autenticada de ngrok

start-ngrok:
	@echo "[+] Iniciando túnel ngrok en puerto $(NGROK_PORT)..."
	@mkdir -p tekton
	@pkill ngrok || true
		@nohup ngrok http $(NGROK_PORT) > tekton/ngrok.log 2>&1 &
	@echo "[...] Esperando que ngrok inicie el túnel..."
	@for i in 1 2 3 4 5 6 7 8 9 10; do \
	  URL=$$(curl --silent http://localhost:4040/api/tunnels | jq -r '.tunnels[0].public_url'); \
	  if [ "$$URL" != "null" ]; then echo $$URL > $(NGROK_URL_FILE); break; fi; \
	  sleep 1; \
	done

	@echo "[✓] ngrok URL capturada:"
	@cat $(NGROK_URL_FILE)

sample-payload:
	@echo "[+] Descargando payload de ejemplo..."
	@mkdir -p tekton
	@curl -s -o $(SAMPLE_PAYLOAD) https://raw.githubusercontent.com/github/docs/main/content/webhooks/event-payloads/push.json

webhook-ngrok:
	@echo "[+] Enviando webhook a través de ngrok..."
	@curl -s -X POST "$$(cat $(NGROK_URL_FILE))" \
	  -H "Content-Type: application/json" \
	  -H "X-GitHub-Event: push" \
	  --data @$(SAMPLE_PAYLOAD)

cleanup-ngrok:
	@echo "[+] Deteniendo ngrok..."
	@pkill ngrok || true
	@rm -f $(NGROK_URL_FILE)

