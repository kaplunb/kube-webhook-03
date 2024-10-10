This is a simple Kubernetes validating admission webhook.

It will allow pod creation and update request if it has a label "required-label:" and some text to it.

Manifest that creates the webhook is scoped to a namespace. Otherwise the webhook becomes cluster scoped, and starts interfering with every pod creation.

Jenkins pipeline is configured to build on kubernetes with kaniko. You will need to change the 'cloud' to match your configuration.
I also use the github registry to store the image and the same access token both for checkout and for the registry. There is an additional manifest to create the github-registry-secret.

It is essential to get the certificates right and cert-manager is expected.

Deployment sequence:
- create a namespace (I use 'webhook-system')
- run the validating-webhook-cert.yaml with your values, it will create a certificate and a secret
- next apply the webhook-infra.yaml to get the conrtoller pod and service running
- finally apply the manifest called validating-webhook-only.yaml

Testing:
- switch to the namespace where the webhook is deployed
- apply the valid-pod.yaml and the invlid-pod.yaml from the test-deployments folder

