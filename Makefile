
all:
	(cd cmd/podmon; make clean build docker push)

check:
	@scripts/check.sh ./internal/monitor ./internal/k8sapi ./internal/csiapi ./internal/criapi ./cmd/podmon  

