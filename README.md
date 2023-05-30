# perunctl

The main goal of this CLI tool is to enable developers to debug their containerized app as if they were debugging it locally inside their favorite IDE (currently, only vscode is supported). The containerized app will run in a local Docker container while communicating with the external container services also running locally.

## Some definitions first ##

**Perun service** - a containerized application.
we support multiple types of services :
- The docker service : its a service representing a docker image where the command and args can be overwritten if need be.
- The local service : a representation of a local folder where the application repository is found, a Dockerfile should already be created at the root of this location, defining the docker container of the application.
  *currently node and python applications are only supported this should be specified in the yaml as well.*
 
**Perun environment** - a grouping of services that make the whole application

**Perun workspace** - a logical grouping of environments

## Environment definition ##
The first step in using the Perun CLI is to define the application in a YAML file. This file will contain the various services of the application and their dependencies. To simplify the process for Kubernetes cluster users, we offer environment import functionality. This feature allows users to connect to a specified Kubernetes namespace and import the running environment as a Perun environment YAML.

Each service in the environment YAML can define its dependencies, which will influence the order of service loading when working in local mode. Additionally, each service can specify its environment variables, which will be injected into the local running process upon loading.

## Pre-requisites

* A running Docker setup on the local machine
* perunctl and events binaries deployed under the same path


## The perunctl CLI
```
Usage:
  perunctl [command]

Available Commands:
  activate    activate Perun environment in a target workspace
  apply       apply the provided env on a workspace, in dry run mode the environment will be analyzed and persisted but not loaded into the target deployment
  deactivate  deactivate Perun environment in a target workspace
  destroy     Destroys and clears given workspace
  generate    generate debug config for supplied service
  help        Help about any command
  import      import target environment into a workspace
  init        initialize empty Perun workspace
  list        list existing perun workspaces

Flags:
      --config string   config file (default is $HOME/.perun.yaml)
  -h, --help            help for perunctl

Use "perunctl [command] --help" for more information about a command.
```

## Basic steps for debugging a service
1. Create the Perun environment YAML file based on a specific schema (refer to the [examples](https://github.com/perun-cloud-inc/perunctl/tree/main/examples))
2. Apply the desired Perun environment locally, which will load all the Docker containers in that environment on your local machine.
```
perunctl init -w <workspace-name>
perunctl apply -w <workspace-name> -e <env-name> -p <path-to-the-env-yml>
```

OR 

1. Import a Kubernetes namespace into a local Perun environment representation. Please check the command for additional supported flags
```
perunctl import -t k8s -w <target-workspace-name> -n <namespace-to-import> -c <k8s-cluster-name>
```
For example, if you have a Kubernetes cluster where [microservices-demo](https://github.com/GoogleCloudPlatform/microservices-demo) is deployed, running the import command on the namespace where it is deployed will generate a workspace YAML under ~/.perun/workspaces/<workspace-name>/ directory. You can refer to the example folder for a reference." 

2. Activate the imported environment... this will locally load all the docker containers in that environment. 
```
perunctl activate -w <workspace-name> -e <env-name>
```

3. generating the debug configuration of the desired service, this will generate vscode launch configuration and persist it under the given repository location of said service.
```
perunctl generate -w <workspace-name> -e <env-name> -s <service-name-to-debug> --source-location <local folder where the service source code resides> --source-type <node or python> --command <the command  to run for loading the application>
```
> :warning: **Please note that any existing vscode configuration will be overwritten**

In the referenced project example [microservices-demo](https://github.com/GoogleCloudPlatform/microservices-demo), to debug the email service, which is written in Python, you can execute the following generate command.

```
perunctl generate -w <workspace-name> -e <env-name> -s emailservice --source-location <local folder where the service source code resides> --source-type python --command "python email_server.py"
```
to debug the payment service which is a nodejs source code, you can run the following generate command.
```
perunctl generate -w <workspace-name> -e <env-name> -s paymentservice --source-location <local folder where the service source code resides> --source-type node --command "node index.js"
```

the above commands will generate vscode launch configuration and persist it in the respective vscode folder of the relevant project specified by the source-location arg.


4. open vscode to the service workspace. you will be able now to run the debug target in the generated launch configuration and start your debugging session. this will effectively stop the original service docker container and load your debug container while routing all traffic to it.
5. once debug session is over all the above re-routing will be reverted and the original container will be back in a running state.

**see more environment examples under the [examples](https://github.com/perun-cloud-inc/perunctl/tree/main/examples) folder**



