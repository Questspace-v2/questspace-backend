dev:
	cd src && \
		go build -o ../questspace.out ./cmd/questspace/main.go && \
		cd ..
	./questspace.out --environment=dev --config=./conf/
