# Subst

A simple extension over kustomize, which allows further variable substitution and introduces simplified yet strong secrets management (for multi tenancy use-cases). Extends to functionality of kustomize for ArgoCD users.

## Functionality

The idea for subst is to act as complementary for kustomize. You can reference additional variables for your environment or from different kustomize paths, which are then accesible across your entire kustomize build. The kustomize you are referencing to is resolved (it's paths). In each of these paths you can create new substitution files, which contain variables or secrets, which then can be used by your kustomization. The final output is your built kustomization with the substitutions made.

By default the all files are considered using this regex `(subst\.yaml|.*(ejson))`. You can change the regex using:

```bash
subst render . --file-regex "custom-values\\.yaml"
```

## Getting Started

For `subst` to work you must already have a functional kustomize build. Even without any extra substitutions you can run:

```bash
subst render <path-to-kustomize>
```

Which will simply build the kustomize.

### ArgoCD

Install it with the [ArgoCD community chart](https://github.com/argoproj/argo-helm/tree/main/charts/argo-cd). These values should work:

```yaml
...
    repoServer:
      enabled: true
      clusterAdminAccess:
        enabled: true
      containerSecurityContext:
        allowPrivilegeEscalation: false
        capabilities:
          drop:
          - all
        readOnlyRootFilesystem: true
        runAsUser: 1001
        runAsGroup: 1001
      volumes:
      - emptyDir: {}
        name: subst-tmp
      - emptyDir: {}
        name: subst-kubeconfig
      extraContainers:
      - name: cmp-subst
        args: [/var/run/argocd/argocd-cmp-server]
        image: ghcr.io/bedag/subst-cmp:v1.0.0
        imagePullPolicy: Always
        securityContext:
          allowPrivilegeEscalation: false
          capabilities:
            drop:
            - all
          readOnlyRootFilesystem: true
          runAsUser: 1001
          runAsGroup: 1001
        resources:
          limits:
            cpu: 500m
            memory: 512Mi
          requests:
            cpu: 100m
            memory: 128Mi
        volumeMounts:
          - mountPath: /var/run/argocd
            name: var-files
          - mountPath: /home/argocd/cmp-server/plugins
            name: plugins
          # Starting with v2.4, do NOT mount the same tmp volume as the repo-server container. The filesystem separation helps
          # mitigate path traversal attacks.
          - mountPath: /tmp
            name: subst-tmp
          - mountPath: /etc/kubernetes/
            name: subst-kubeconfig
...
```

Change version accordingly.

### Available Substitutions

You can display which substitutions are available for a kustomize build by running:

```bash
subst substitutions .
```

See available options with:

```bash
subst substitutions -h
```

### Paths

The priority is used from the kustomize declartion. First all the patch paths are read. Then the `resources` are added in given order. So if you want to overwrite something (highest resource), it should be the last entry in the `resources` The directory the kustomization is recursively resolved from has always highest priority.

See example `/test/build/kustomization.yaml`

```bash
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
  - operators/
  - ../addons/values/high-available
patches:
  - path: ../../apps/common/patches/argo-appproject.yaml
    target:
      kind: AppProject
  - path: ./patches/argo-app-settings.yaml
    target:
      kind: Application
```

Results in the following paths (order by precedence):

  1. /test/build/
  2. /test/build/../addons/values/high-available
  3. /test/build/operators/
  4. /test/build/patches
  5. /test/build/../../apps/common/patches

Note that directories do not resolve by recursion (eg. `/test/build/` only collects files and skips any subdirectories).

### Environment

For environment variables which come from an argo application (`^ARGOCD_ENV_`) we remove the `ARGOCD_ENV_` and they are then available in your substitutions without the `ARGOCD_ENV_` prefix. This way they have the same name you have given them on the application ([Read More](https://argo-cd.readthedocs.io/en/stable/operator-manual/config-management-plugins/#using-environment-variables-in-your-plugin)). All the substitutions are available as flat key, so where needed you can use environment substitution.

## Spruce

[Spruce](https://github.com/geofffranks/spruce) is used to access the substitution variables, it has more flexability than envsubst. You can grab values from the available substitutions using [Spruce Operators](https://github.com/geofffranks/spruce/blob/main/doc/operators.md). Spurce is great, because it's operators are valid YAML which allows to build the kustomize without any further hacking.

## Secrets

You can both encrypt files which are part of the kustomize build or which are used for substitution. Currently for secret decryption we support [ejson](https://github.com/Shopify/ejson). The principal for the decryption provider is, that it should load the private keys while a substitution build is made instead of having a permanent keystore. This allows for secret tenancy (eg. one secret per argo application). The private keys are loaded from kubernetes secrets, therefor the plugin also creates it's own kubeconfig.

The secrets are loaded based on how the environment variables `$ARGOCD_APP_NAME` and `$ARGOCD_APP_NAMESPACE` are used. If an application is in a project, the value of `$ARGOCD_APP_NAME` looks like this: `<project-name>_<application-name>`. For example, if the application `my-app` is in the project `my-project`, the value of `$ARGOCD_APP_NAME` is `my-project_my-app`. All special characters within are converted to `-` (dash). For example, if the application `my-app` is in the project `my-project`, the value of `$ARGOCD_APP_NAME` is `my-project-my-app`. So the secret reference is then `my-project-my-app` in the secret namespace (Assuming `--convert-secret-name=false`).

By default the `--convert-secret-name` is enabled. This removes the project prefix from the secret. If you create an application `test` in the namespace `test-reserved` the plugin is looking for private keys in the  secret `test` in the namespace `test-reserved`.

The values for the secret name and namespace can also be set explicitly, however this way you lose the multi-tenancy aspect of the secrets management:

```bash
subst render --secret-name static-name --secret-namespace static-namespace .
```

You can disable the lookup of the private keys in Kubernetes secrets. This is useful if you want to use the substition without access to the kubernetes clusters. The decryption providers allow to enter the private keys directly. This is useful for CI/CD pipelines or local testing (See decryption provider documentation).

```bash
subst render . --skip-secret-lookup
```

Decryption can be disabled, in that case the files are just loaded, without their encryption properties (might be useful if you dont have access to the private keys to decrypt the secrets):

```bash
subst render . --skip-decrypt
```

See below how to work with the different decryption providers.

### EJSON

[EJSON](https://github.com/Shopify/ejson) allows simple secrets management.

You can encrypt entire files using EJSON. The file must be in JSON format. The entire file will be encrypted, which may not be useful in all cases.

#### Local installation

##### Go

```bash
go install github.com/Shopify/ejson/cmd/ejson@v1.5.3
```

##### Brew

Unfortunately, the [Brew](https://github.com/Shopify/ejson?tab=readme-ov-file#installation) package is unmaintained. We therefore don't recommend this installation method.

##### Kubernetes

For all decryptors you can create a Kubernetes secret, which contains the private information for secret decryption.

## Installation

### Go

```bash
go install github.com/bedag/subst/subst@v1.0.0
```

### Docker

```bash
docker run --rm -it ghcr.io/bedag/subst:v1.0.0 -h
```

### Github releases

[github.com/bedag/subst/releases](https://github.com/bedag/subst/releases)
