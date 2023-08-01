# k2d 

## What is k2d?

K2D was created to solve a very specific problem; enabling the use of Kubernetes primitives on the resource-constrained compute devices that underpin Industrial IoT use cases. 

Industrial IoT deployments generally span three distinct environments; Datacenter, Regional/Plant Comms Rooms, and Devices. 

Datacenter deployments house the applications that run the overall processes that underpin the software solution, eg Manufacturing Execution System (MES), an Overall Equipment Efficiency Dashboard, Integrated Quality Management System, Digital Twin Software, etc.

Regional/Plant deployments house applications that aggregate metrics to/from the multitude of devices within their physical proximity, eg MQTT Brokers, Time Series Databases, Protocol Routers. 
Devices (industrial compute or automation controllers) run the software that interfaces with machinery, via Programmable Logic Controllers (PLC's) using either open or proprietary protocols. Here common software is Node-Red, MQTT, OPC/MODBUS/MQTT Transformers, OPC Routers etc. 

Kubernetes is now prevalent in the Datacenter, driven predominately by IT teams that hold the technical foundations and competence to support and operate it. Kubernetes is also regularly deployed in regional/plant environments because these are often also managed by a central IT team (but not always). So far though, it's been near-impossible to run Kubernetes on the devices that connect to plant and machinery.

Kubernetes is an amazing technology; it is a universal language (and API) that defines how applications should be run (declaratively), regardless of where they run, or how the underlying platform is configured. It is the first time that we have had the ability to define a common software "manifest" and that manifest will result in an application deployment that is configured and operates exactly the same way, every single time, everywhere.

The problem holding back Kubernetes adoption on device deployments is down to hardware resource constraints, and operational manageability concerns.

Kubernetes can only run on devices with sufficient CPU, RAM, and Disk, and it is challenging to operate Kubernetes unless you are operationally trained in the technology. 

So what happens when you have the desire to standardize on Kubernetes manifests as your deployment language, but your devices and OT engineering teams are unable to accommodate? 

K2D.

K2D merges two worlds... it allows OT engineering teams to interact with their running applications and devices using the incredibly simple Docker UX, whilst allowing IT operations teams to interact with the very same devices using Kubernetes tooling, and to deploy applications to these devices using the Kubernetes manifest format. 

K2D allows extremely resource-constrained devices to be able to accept management operations via a stripped-down Kubernetes API, negating the need to run Kubernetes on the device itself.

## How does K2D work?

k2d is a single container that runs on a Docker (or Podman) Host, which listens on https port 6443 for a limited number of Kubernetes API calls. When the container receives these Kubernetes API calls, k2d parses and translates them into Docker API instructions, which it then executes on the underlying Docker Host. 

As a result, the translator allows Linux enabled devices, with as little as 1x 700Mhz ARM32 CPU, 512MB of RAM and a 16GB SD-Card, to be managed as if they were single-node Kubernetes environments.

The translator is highly resource efficient, requiring CPU cycles only when it's actively translating commands, uses just 20MB of RAM (on top of the OS and Docker requirement of ~120MB), and produces negligible disk IO. Even a device with 512MB of RAM would have ~370MB of RAM available for running applications!  

As k2d is a translator, the devices running it are NOT actually running Kubernetes, so there are NO Kubernetes components to manage... no etcd, no KubeDNS, nothing... just Docker (or Podman). 

This makes day 2 operations easier too. All you need is docker knowledge to manage the devices. Of course, as the device is not running Kubernetes, and API instructions are being translated, only a small subset of Kubernetes capability is available. However what is available in K2D is aligned with the intended use case, being deployment of applications on IIOT devices.

The API translations all happen in real-time, so you are able to interact with k2d using Kubernetes native tooling (KubeCTL, K9s, ArgoCD, HELM, Portainer), and you should not see any difference in behavior.
As an additional benefit, as the translations are bi-directional, any Docker management commands executed outside of K2D on the docker host directly, are also translated and appear as Kubernetes resources when later inspected via Kubernetes tooling through the translator.

‍Note that by design, not all API commands are implemented. If k2d does not support a command, it will silently fail, so as not to break Kubernetes tools that might interfere with the translator.

For more information, see the [k2d documentation](https://portainer-1.gitbook.io/0.1.0-alpha/).
