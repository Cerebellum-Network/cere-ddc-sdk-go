protobuf:
	docker run --rm       									\
		-v ${PWD}/ddc-schemas/storage/protobuf:/proto_path	\
		-v ${PWD}/model:/go_out                             \
		rvolosatovs/protoc:3.3                             	\
			--experimental_allow_proto3_optional            \
			--proto_path=/proto_path   						\
			--go_out=/go_out     							\
			$$(find ddc-schemas/storage/protobuf -name '*.proto' -printf '%P ')

fix-docker-mess:
	sudo chown $$(whoami) -R model/pb/