# kron
[![Build Status](https://travis-ci.org/sabakaio/kron.svg?branch=master)](https://travis-ci.org/sabakaio/kron)

## Usage
First, you need to create a service account for kron:
```yaml
# account.yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: kron
```

```bash
kubectl create -f account.yaml
```

Then, create a job with labels and annotations for kron:
```yaml
# job.yaml
apiVersion: batch/v1
kind: Job
metadata:
  name: "sample-job"
  labels:
    kron: true
  annotations:
    schedule: "@every 5m"
spec:
  completions: 0
  template:
    metadata:
      name: "sample-job"
    spec:
      containers:
      - image: alpine:3.4
        name: "sample-job"
        command: ["echo", "$TEST_VAR"]
        env:
        - name: TEST_VAR
          value: ok
      restartPolicy: Never
```

```bash
kubectl create -f job.yaml
```

Set `spec.completions` to `0` if you don't want the job to be executed when you create it.

Finally, run the kron service:
```yaml
apiVersion: v1
kind: Pod
metadata:
  name: kron
spec:
  serviceAccountName: kron
  containers:
  - image: "sabaka/kron"
    name: kron
```

## Development

```bash
glide install
```
