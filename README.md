# Dynu Webhook for cert-manager

[![GitHub](https://img.shields.io/badge/github-blackdark%2Fcert--manager--webhook--dynu-blue?logo=github)](https://github.com/blackdark/cert-manager-webhook-dynu)

A [cert-manager][1] webhook to solve an ACME DNS01 challenge using the [Dynu][2] DNS API.

Source: [github.com/blackdark/cert-manager-webhook-dynu](https://github.com/blackdark/cert-manager-webhook-dynu)

## Prerequisites

A Kubernetes cluster with cert-manager deployed. If you haven't already installed cert-manager, follow the guide [here][3].

## Deployment

### Using regular manifests

An example webhook deployment, with its associated roles, role bindings, service and required TLS certificates, is provided in the file `examples/deploy/dynu-webhook.yaml`. The example manifest will deploy to the `cert-manager` namespace. To set a different namespace, replace all instances of `cert-manager` with the namespace of your choice. Don't miss the annotation `cert-manager.io/inject-ca-from` in the `APIService` or the `dnsNames` for the webhook `Certificate`. The following will replace all the necessary namespace references in the example manifest:

```bash
$ sed -i 's/cert-manager/yournamespace/g' examples/deploy/dynu-webhook.yaml
```

Once you're satisfied with the changes, deploy the webhook with:

```bash
$ kubectl apply -f examples/deploy/dynu-webhook.yaml
```

### Using Helm (from this repository)

```bash
$ helm upgrade --install dynu-webhook ./deploy/dynu-webhook \
  --namespace cert-manager \
  --set groupName=acme.example.com \
  --set certManager.namespace=cert-manager \
  --set certManager.serviceAccountName=cert-manager
```

If you want to scope secret access, set `secretsNames`:

```bash
$ helm upgrade --install dynu-webhook ./deploy/dynu-webhook \
  --namespace cert-manager \
  --set groupName=acme.example.com \
  --set certManager.namespace=cert-manager \
  --set certManager.serviceAccountName=cert-manager \
  --set-json 'secretsNames=["dynu-api-key"]'
```

### Using Helm (OCI chart from GHCR)

```bash
$ helm upgrade --install dynu-webhook oci://ghcr.io/blackdark/charts/dynu-webhook \
  --version 1.2.3 \
  --namespace cert-manager \
  --set groupName=acme.example.com \
  --set certManager.namespace=cert-manager \
  --set certManager.serviceAccountName=cert-manager
```

If you want to scope secret access, set `secretsNames`:

```bash
$ helm upgrade --install dynu-webhook oci://ghcr.io/blackdark/charts/dynu-webhook \
  --version 1.2.3 \
  --namespace cert-manager \
  --set groupName=acme.example.com \
  --set certManager.namespace=cert-manager \
  --set certManager.serviceAccountName=cert-manager \
  --set-json 'secretsNames=["dynu-api-key"]'
```

## Usage

### Deploy an API Key Secret

The Dynu API key needs to be placed into a Kubernetes secret. You can use the file `examples/dynu-api-key.yaml` as a starting point. Place your Base64-encoded API key into the manifest. This can be obtained with:

```bash
$ echo -n "your-api-key" | base64
```

The API key can be created in the Dynu control panel under **API Credentials**.

### Deploy an Issuer

An example `ClusterIssuer` is provided in the `examples/letsencrypt-staging-issuer.yaml` file. It uses the Let's Encrypt staging server, but it can be adapted for the Let's Encrypt production server or other ACME servers. `Issuer` can be used instead of `ClusterIssuer` if you only want to issue certificates in a single namespace.

### Deploy a Certificate

An example certificate manifest is provided in `examples/test-certificate.yaml`. Edit as required for production certificates.

### Check the Certificate

Use the following to check the deployed certificate, replacing _test-example-com-tls_ with the **secretName** you used in the previous step.

```bash
$ kubectl get secret test-example-com-tls -o json | jq -r '.data."tls.crt"' | base64 -d | openssl x509 -text -noout
```

## Building

```bash
$ make build
```

## Releasing Image and Helm Chart

The repository includes:

- `.github/workflows/release-image.yaml` for the container image
- `.github/workflows/release-helm-chart.yaml` for the Helm chart (OCI in GHCR)

Push a tag like `v1.2.3` to publish:

- Multi-arch image (`linux/amd64`, `linux/arm64`) to `ghcr.io/blackdark/cert-manager-webhook-dynu`
- Helm chart to `oci://ghcr.io/blackdark/charts/dynu-webhook` with chart version `1.2.3`

You can also run workflows manually from the Actions tab (`workflow_dispatch`).

For manual chart releases, pass `chart_version` (for example `1.2.3`) when starting the `release-helm-chart` workflow.

### Running the test suite

All DNS providers **must** run the DNS01 provider conformance testing suite,
else they will have undetermined behaviour when used with cert-manager.

See [here][4] for testing details.

[1]: https://github.com/cert-manager/cert-manager
[2]: https://www.dynu.com/
[3]: https://cert-manager.io/docs/installation/kubernetes/
[4]: ./testdata/dynu/README.md
