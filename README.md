![Release](https://github.com/Pennsieve/pennsieve-agent/actions/workflows/go.yml/badge.svg?branch=main)
![Version](https://img.shields.io/github/v/release/Pennsieve/pennsieve-agent?include_prereleases)

# pennsieve-agent
Pennsieve Agent and Command Line Interface..


## Features

1. Command Line Interface (leveraging Cobra and Viper)
2. Local SQLite database for userInfo storage and session caching
3. Integration with the Pennsieve-Go Library
4. gRPC server for handling large tasks such as uploading data


## Installing the Pennsieve Agent

### Using the Installlers (recommended)

Download the latest installer for your operating system: https://github.com/Pennsieve/pennsieve-agent/releases

### From Source

1. Clone the Pennsieve Agent repository
2. Install Golang on the machine
3. Run `go build` or `go install` in the pennsieve-agent folder
4. Symlink "pennsieve" to the output (the linux executable is called 'pennsieve-agent' instead of 'pennsieve')




## Releasing a new version

1. Merge updates into the main branch
2. Create a new tag in main and name the tag: vx.x.x following [semantic versioning](https://semver.org/).

    e.g ```git tag -a 0.0.1 -m "Initial release"```

    Given a version number MAJOR.MINOR.PATCH, increment the:

    - MAJOR version when you make incompatible API changes,
    - MINOR version when you add functionality in a backwards compatible manner, and
    - PATCH version when you make backwards compatible bug fixes.

3. Push the tag to Gihub

    eg. ```git push origin 0.0.1```
    
This will trigger Github Actions to create a new release with the same name.


## Building the GRPC Protobuf 
The gRPC server is defined in the ```api/v1/agent.proto``` file. Use the following command to generate the go structs, GRPC client and server interfaces: 

```shell
make compile
```

for Python, use:
```shell
make compile-python
```

## Pennsieve Configuration File
The CLI depends on a configuration file in the ~/.pennsieve folder. You can initialize this file 
with the ```pennsieve-agent config wizard``` command. 

If you are using a profile for a non-standard environment, you can manually add the following key/values to the configuration file:

```shell
api_host = XXXX (eg. https://api.pennsieve.net)
upload_bucket = XXXXX (eg. pennsieve-dev-uploads-v2-use1)
```

## Configuration with Environment variables
You can set agent configuration parameters by updating the configuration file or by setting the following Environment Variables:

- PENNSIEVE_AGENT_PORT
- PENNSIEVE_AGENT_UPLOAD_WORKERS
- PENNSIEVE_AGENT_CHUNK_SIZE

You can use environment variables to set your profile using the following variables

- PENNSIEVE_API_KEY
- PENNSIEVE_API_SECRET
- PENNSIEVE_UPLOAD_BUCKET (optional)
- PENNSIEVE_API_HOST (optional)

If you set the PENNSIEVE_API_KEY, the agent will not use the configuration file and use the profile specified in the environment variables. Note that upload bucket and api_host are optional and default to the production version of the platform.

### Specifying Agent Parameters
The Pennsieve Agent is configured with a set of default parameters. You can update these parameters by specifying these in the configuration file. Specifically, you can update:

```shell
[agent]
upload_chunck_size:     The size of each chunk that is uploaded to the platform as part of a multipart upload process
port                    The port on which the agent is available
upload_workers          The number of files that are uploaded simultaneously.
```

## Logging
We are using the [logrus](https://github.com/sirupsen/logrus) library for logging.

## Testing
We are using the [testify](https://github.com/stretchr/testify) package for unit testing Golang code. 

The goal is to keep testing simple and effective. There is no need to make testing itself complex. 
