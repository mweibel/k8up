# Dev Environment
You'll need:

* Minishift or Minikube
* golang installed :) (everything is tested with 1.10.1)
* dep installed
* Your favorite IDE (with a golang plugin)
* docker
* make

## Generate kubernetes code
If you make changes to the CRD struct you'll need to run code generation. This can be done with make:

```
cd /project/root
make generate
```

This creates the client folder and deepcopy functions for the structs. This needs to be run on a local docker instance so it can mount the code to the container.

## Run the operator in dev mode

```
cd /to/go/project
minishift start
oc login -u system:admin # default developer account doesn't have the rights to create a crd
#The operator has the be run at least once before to create the CRD
go run cmd/operator/*.go -development
#Add a demo backupworker (adjust the variables to your liking first)
kubectl apply -f manifest-examples/baas.yaml
#Add a demo PVC if necessary
kubectl apply -f manifest-examples/pvc.yaml
```

## Build and push the Restic container
The container has to exist on the registry in order for the operator to find the correct one.

```
minishift start
oc login -u developer
eval $(minishift docker-env)
docker login -u developer -p $(oc whoami -t) $(minishift openshift registry)
cd cmd/restic
docker build -t $(minishift openshift registry)/myproject/test .
docker push $(minishift openshift registry)/myproject/test
```

## Example resource
```yaml
apiVersion: appuio.ch/v1alpha1
kind: Backup
metadata:
  namespace: baas-test
  name: baas-test
spec:
  dryRun: true # Not used yet
  schedule: "* * * * *" #every minute
  checkSchedule: "* * * * *" # When the checks should run default once a week
  keepJobs: 4 # How many job objects should be kept to check logs
  backend:
    password: asdf # The restic encryption password
    s3: # Self explaining
      endpoint: http://10.144.1.133:9000
      bucket: baas
      username: 8U0UDNYPNUDTUS1LIAF3
      password: ip3cdrkXcHmH4S7if7erKPNoxDn27V0vrg6CHHem
  retention: # Default 14 days
    keepLast: 2 # Absolute amount of snapshots to keep overwrites all other settings
    keepDaily: 0
    # Available retention settings:
    # keepLast
    # keepHourly
    # keepDaily
    # keepWeekly
    # keepMonthly
    # keepYearly
    # keepTags # Not yet implemented
```
# Deploy and Configure the Operator
To deploy the operator you'll need to adjust some config in the manifest folder. The contents of that folder:
* `baas-example.yaml` an example backup
* `operator.yaml` the actual operator
* `pv-example.yaml` example for a pv
* `pvc-example.yaml` example for a pvc
* `role-bindings.yaml` cluster wide permissions necessary
* `service-account.yaml` the service account for the permissions

## Configuration
Various things can be configured via environment variables:
* `BACKUP_IMAGE` URL for the restic image, default: `172.30.1.1:5000/myproject/restic`
* `BACKUP_ANNOTATION` the annotation to be used for filtering, default: `appuio.ch/backup`
* `BACKUP_CHECKSCHEDULE` the default check schedule, default: `0 0 * * 0`
* `BACKUP_PODFILTER` the filter used to find the backup pods, default: `backupPod=true`
* `BACKUP_DATAPATH` where the PVCs should get mounted in the container, default `/data`
* `BACKUP_JOBNAME` names for the backup job objects in OpenShift, default: `backupjob`
* `BACKUP_PODNAME` names for the backup pod objects in OpenShift, default: `backupjob-pod`
* `BACKUP_RESTARTPOLICY` set the RestartPolicy for the backup jobs, default: `OnFailure`

You only need to adjust `BACKUP_IMAGE` everything else can be left default.

## Installation
After everything is set to your liking in the yaml files you can deploy it with:

```bash
kubectl apply -f manifest/service-account.yaml
kubectl apply -f manifest/role-bindings.yaml
kubectl apply -f manifest/operator.yaml
# and then create a backup
kubectl apply -f manifest/baas-exampler.yaml
```

You may need to adjust the namespace in `service-account.yaml` and `role-bindings.yaml`.

Please see the example resource here in the readme for an explanation of the various settings.