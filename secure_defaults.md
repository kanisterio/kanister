# Improving Job Pod Security Defaults in Kanister

## Overview:
In Kanister's current behavior, job pods created within the Kanister controller's namespace automatically inherit the controller's service account. While this simplifies execution, it also broadens the permissions granted to job pods, potentially increasing the risk of privilege escalation and introducing security vulnerabilities.

## Secure Defaults:
To enhance security within the Kanister environment, it is strongly recommended to explicitly set the `serviceAccountName` to a dedicated least-privileged service account for job pods. This approach ensures that job pods operate with minimal permissions, tailored specifically to their operational requirements. Furthermore, configuring `automountServiceAccountToken` to `false` by default is advised. This prevents the automatic mounting of service account tokens, significantly reducing the risk of unauthorized access or privilege escalation.

By adopting these secure defaults, Kanister enforces the principle of least privilege for all job pods, thereby minimizing their attack surface and bolstering overall security. These settings will be applied to all job pods by default, ensuring a consistent and secure operational environment.

Additionally, users can enable these secure defaults explicitly by setting the Helm flag `secureDefaultsForJobPods` to true. When this flag is activated, Kanister will automatically apply the recommended configurations, further simplifying the process of securing job pods.

## Configuring RBAC and Service Account Token Mounting for Job Pods:
Secure defaults enforce a least-privileged environment for job pods, which may lead to failures if the necessary permissions are not configured. To address this, users can customize the RBAC settings and security configurations using the `podOverride` option. This feature allows users to specify a custom `serviceAccountName` and control the automatic mounting of the service account token by setting `automountServiceAccountToken` to `true` or `false`.

### Example Configuration:
```yaml
actions:
  backup:
    phases:
      - func: func_name
        name: phase_name
        podOverride:
          serviceAccountName: provided_svc_name
          automountServiceAccountToken: true
```

### Steps to Customize:
1. **Define a Service Account**: Create a dedicated service account (`provided_svc_name`) in the namespace where the job pod will execute.
2. **Configure RBAC**: Assign roles and role bindings to the service account, ensuring permissions are limited to the job pod's operational needs.
3. **Apply Overrides**: Use the `podOverride` option in the Kanister action definition to specify the service account and token mounting preferences.

By following these steps, users can ensure that job pods operate securely with tailored permissions and token management.

