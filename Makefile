.PHONY: compile

compile:
	@echo "Compiling GRPC definitions"
	protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative api/v1/agent.proto

compile_python:
	@echo "Compiling GRPC definitions for Python"
	python -m grpc_tools.protoc --python_out=build/gen/ -I. --grpc_python_out=build/gen api/v1/agent.proto

release:
	@ echo ""
	@build=$$(git tag | sort -n -r | head -n 1 |  awk -F_ '{print $$1}'); \
	build=$$((build+1)); \
	commit=$$(git log -1 --pretty=format:%h); \
	version=$${build}_$${commit}; \
	echo Version: $$version;
	@echo "\nChangelog"
	@git log --format="%h %s" $$(git tag | sort -n -r | head -n 1 | awk -F_ '{print $$2}').. | sed -e 's/^/-\ /g'
