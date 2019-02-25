## Yummy

Yummy is a dynamic local volume provisioner based on lvm. It creates lvm according to pvc and mount it to node, thus the [external storage](https://github.com/kubernetes-incubator/external-storage) local volume provisioner will detect the new mount point and create pv.

[Demo](https://www.youtube.com/watch?v=DB3u1PeOL8s)

## build

### scheduler

```bash
./build-scheduler.sh
```

Output file in `cmd/scheduler`

### agent

```bash
./build-agent.sh
```

Output file in `cmd/agent`

## run

### run scheduler as master

```bash
#!/bin/bash
KUBECONFIG=/home/silenceshell/.kube/config ./cmd/scheduler/scheduler
```

### run agent on all nodes who provision lvm

1 create agent config: /etc/yummy/config/agentConfigMap

```yaml
agentConfigMap:
  volumeGroup: volume-group1
  mountDir: /data/local
```

`volumeGroup` is the vg of this node; `mountDir` is the dir that new lv will be mounted.

2 run agent

```bash
#!/bin/bash
KUBECONFIG=/home/silenceshell/.kube/config MY_NODE_NAME=ubuntu-2  ./cmd/scheduler/agent
```

## Known issue

Pod should be deleted first, or the related lv maybe count not be removed. Will fix it later.
