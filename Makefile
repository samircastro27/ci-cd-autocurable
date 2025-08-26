.PHONY: build push deploy operator

build:
	docker build -t localhost:32000/demo-microservice:latest ./microservice

push:
	docker push localhost:32000/demo-microservice:latest

deploy:
	kubectl apply -f microservice/k8s

tekton:
	kubectl apply -f tekton/pipeline.yaml
	kubectl apply -f tekton/triggers.yaml

operator:
	kubectl apply -f operator/config/crd.yaml
	kubectl apply -f operator/config/samples/healingpolicy.yaml

monitoring:
	kubectl apply -f monitoring/prometheus.yaml
	kubectl apply -f monitoring/grafana.yaml

all: build push deploy tekton operator monitoring

