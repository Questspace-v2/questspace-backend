dev:
	cd src && \
		go build -o ../questspace.out ./cmd/questspace/main.go && \
		cd ..
	ENVIRONMENT=dev ./questspace.out --config=./conf/
