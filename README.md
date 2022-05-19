# pennsieve-agent
Second iteration of the Pennsieve Agent


## Features

1. Command Line Interface (leveraging Cobra and Viper)
2. Local SQLite database for userInfo storage and session caching
3. Integration with the Pennsieve-Go Library
4. gRPC server for handling large tasks such as uploading data


## Releasing a new version

1. Merge updates into the main branch
2. Create a new tag in main and name the tag: vx.x.x following [semantic versioning](https://semver.org/).

    e.g ```git tag -a v0.0.1 -m "Initial release"```

    Given a version number MAJOR.MINOR.PATCH, increment the:

    - MAJOR version when you make incompatible API changes,
    - MINOR version when you add functionality in a backwards compatible manner, and
    - PATCH version when you make backwards compatible bug fixes.

3. Push the tag to Gihub

    eg. ```git push origin v0.0.1```
    
This will trigger Github Actions to create a new release with the same name.


## Building the GRPC Protobuf 
The gRPC server is defined in the ```protos/agent.proto``` file. Use the following command to generate the go structs, GRPC client and server interfaces: 

``protoc --go_out=. --go_opt=paths=source_relative \
--go-grpc_out=. --go-grpc_opt=paths=source_relative \
protos/agent.proto
``

for Python, use:
``` python -m grpc_tools.protoc --python_out=build/gen/ -I. --grpc_python_out=build/gen protos/agent.proto```
