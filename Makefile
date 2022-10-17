.PHONY: compile

compile:
	@echo "Compiling GRPC definitions"
	protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative api/v1/agent.proto

compile_python:
	@echo "Compiling GRPC definitions for Python"
	python -m grpc_tools.protoc --python_out=build/gen/ -I. --grpc_python_out=build/gen api/v1/agent.proto