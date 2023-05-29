# perunctl

This CLI tool main goal is to enable developers to debug their containerized app as if they are debugging it locally inside their favorite IDE (currently only vscode is supported).
the containerized app will run in a local docker container while communicating to the external container services running locally as well.

## Some definitions first ##

**Perun service** - a containerized application.
we support multiple types of services :
- The docker service : its a service representing a docker image where the command and args can be overwritten if need be.
- The local service : a representation of a local folder where the application repository is found, a Dockerfile should already be created at the root of this location, defining the docker container of the application.
  *currently node and python applications are only supported this should be specified in the yaml as well.*
 
**Perun environment** - a grouping of services that make the whole application

**Perun workspace** - a logical grouping of environments

## Environment definition ##
The first step of using perun cli is to define the application in a yaml file, the file will contain the various services of the application and their dependencies.
To make things easier for k8s cluster users, we allow environment import which will connect to the specified k8s namespace and import the environment running there as a perun environment yaml.

each service in the environment yaml can define its dependencies influencing the order of the services load when working in local mode.
In addition, each service can define its environment variables which will be injected into the running local process when loaded.

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
1. Creating the perun environment yaml according to a specific schema (see examples)
2. Applying the desired perun environment locally... this will locally load all the docker containers in that environment.
```
perunctl init -w <workspace-name>
perunctl apply -w <workspace-name> -e <env-name> -p <path-to-the-env-yml>
```

OR 

1. import a k8s namespace to a local perun environment representation (check the command for more supported flags)
```
perunctl import -t k8s -w <target-workspace-name> -n <namespace-to-import> -c <k8s-cluster-name>
```
For example if you have a k8s cluster where "https://github.com/GoogleCloudPlatform/microservices-demo" is deployed, running the import command on the namespace where it is deployed will generate a workspace yaml under ~/.perun/workspaces/<workspace-name>/ see example folder for a reference. 
2. Activate the imported environment... this will locally load all the docker containers in that environment. 
```
perunctl activate -w <workspace-name> -e <env-name>
```

3. generating the debug configuration of the desired service, this will generate vscode launch configuration and persist it under the given repository location of said service.
```
perunctl generate -w <workspace-name> -e <env-name> -s <service-name-to-debug> --source-location <local folder where the service source code resides> --source-type <node or python> --command <the command  to run for loading the application>
```
:warning: **Please note that any existing vscode configuration will be overwritten**
in the referenced project example "https://github.com/GoogleCloudPlatform/microservices-demo",
to debug the email service which is a python source code, you can run the following generate command.
```
perunctl generate -w <workspace-name> -e <env-name> -s emailservice --source-location <local folder where the service source code resides> --source-type python --command "python email_server.py"
```
to debug the payment service which is a nodejs source code, you can run the following generate command.
```
perunctl generate -w <workspace-name> -e <env-name> -s paymentservice --source-location <local folder where the service source code resides> --source-type node --command "node index.js"
```

the aboce commands will generate vscode launch configuration and persist it in the respective vscode folder of the relevant project specified by the source-location arg.


4. open vscode to the service workspace. you will be able now to run the debug target in the generated launch configuration and start your debugging session. this will effectively stop the original service docker container and load your debug container while routing all traffic to it.
5. once debug session is over all the above re-routing will be reverted and the original container will be back in a running state.

**see more environment examples under the *examples* folder**



