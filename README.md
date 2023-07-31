# k2d 

## What is k2d?
k2d is a container that runs on a Docker Host, which listens on port 6443 for Kubernetes API calls. When the container receives Kubernetes API calls, k2d parses and translates them into Docker API instructions, which it executes on the underlying Docker Host.

This allows Linux enabled devices to run JUST the docker engine (or podman) and the translator container (which uses just 20MB of RAM) to enable the Docker instance to understand and act upon Kubernetes API calls.

For example, you can request a pod's deployment, and k2d will parse it into the corresponding docker run command. You can request a deployment, and k2d will parse this into multiple docker run commands. You can request the publishing of a service, and k2d will create a docker network and publish the container on that network. Even request the list of all running pods, and k2d will be able to translate appropriately.

**‍Note that not all API commands are implemented*. If k2d does not support a command, it will silently fail not to break Kubernetes tools that might interfere with the translator.

For more information, see the [k2d documentation](https://portainer-1.gitbook.io/0.1.0-alpha/).