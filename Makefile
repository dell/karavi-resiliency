
all:
	(cd cmd/podmon; make clean build docker push)
