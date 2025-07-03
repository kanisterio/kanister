# Improving Job Pod Security Defaults in Kanister

## Overview:
In Kanister's earlier default behavior, job pods created within the Kanister controller's namespace would automatically inherit the controller's service account. While this approach streamlined execution, it inadvertently expanded the scope of permissions granted to job pods, increasing the risk of privilege escalation and potential security vulnerabilities.

## Secure Defaults:
To strengthen security, it is advisable to explicitly set the `serviceAccountName` to a dedicated least-privileged service account for job pods. This ensures that job pods operate with restricted permissions tailored to their specific needs. Additionally, it is recommended to configure `automountServiceAccountToken` to `false` by default. This prevents the automatic mounting of service account tokens, mitigating the risk of unauthorized access or privilege escalation.

By adopting these secure defaults, all job pods will adhere to the principle of least privilege, reducing their attack surface and enhancing overall security within the Kanister environment.

All job pods will be using above setting by default. 

## Configuring RBAC and Service Account Token Mounting for Job Pods
To customize the security settings of job pods, users can leverage the `podOverride` option. This allows specifying a custom `serviceAccountName` and enabling or disabling the automatic mounting of the service account token using `automountServiceAccountToken`. Below is an example configuration:

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

### Steps to Implement:
1. **Create a Service Account**: Define the `provided_svc_name` service account in the namespace where the job pod will run.
2. **Set Up RBAC**: Create roles and role bindings to grant the necessary permissions to the service account. Ensure the permissions adhere to the principle of least privilege.
3. **Apply the Configuration**: Use the `podOverride` option in your Kanister action definition to specify the service account and token mounting settings.

By following these steps, users can ensure that job pods operate securely with tailored permissions and token management.

