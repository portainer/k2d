# k2d (k2d)

WIP

# Development

Install air:

```
go install github.com/cosmtrek/air@latest
```

# Run it

Deploy it:
```
docker run -d --network host -e ADVERTISE_ADDR=10.114.0.2 -v /var/run/docker.sock:/var/run/docker.sock -v /var/lib/k2d:/var/lib/k2d portainer/k2d
```

Get the kubeconfig:
```
mkdir -pv  ~/.kube/
curl --insecure https://localhost:6443/k2d/kubeconfig > ~/.kube/config
```

Use it:
```
kubectl get pods
```

# How it works

## Configmaps and secrets

Stored under /var/lib/k2d/configmaps and /var/lib/k2d/secrets

# Limitations

* ConfigMaps can only hold data, not binary data (however not restricted to 1MB max size anymore, tbc)
* Only supported secret type is Opaque