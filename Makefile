packages = ./model/... ./core/... ./contract/... ./contentadrstorage/...

test:
	go test ${packages}

lint:
	docker run --rm -v ${PWD}:/app -w /app golangci/golangci-lint:v1.45.2 golangci-lint run -v ${packages}

protobuf:
	docker run --rm       									\
		-v ${PWD}/ddc-schemas/storage/protobuf:/proto_path	\
		-v ${PWD}/model:/go_out                             \
		rvolosatovs/protoc:3.3                             	\
			--experimental_allow_proto3_optional            \
			--proto_path=/proto_path   						\
			--go_out=/go_out     							\
			$$(find ddc-schemas/storage/protobuf -name '*.proto' -type f | sed 's/.*\///')

protoc:
	protoc --proto_path=./ddc-schemas/storage/protobuf --go_out=./model ./ddc-schemas/storage/**/*.proto

fix-docker-mess:
	sudo chown $$(whoami) -R model/pb/
