# Troubleshooting

If an ActionSet fails to perform an action, then the failure events can
be seen in the respective ActionSet as well as its associated Blueprint
by using the following commands:

``` bash
# Example of failure events in an ActionSet:
$ kubectl --namespace kanister describe actionset <ActionSet Name>
Events:
  Type     Reason                          Age   From                 Message
  ----     ------                          ----  ----                 -------
  Normal   Started Action                  14s   Kanister Controller  Executing action delete
  Normal   Started Phase                   14s   Kanister Controller  Executing phase deleteFromS3
  Warning  ActionSetFailed Action: delete  13s   Kanister Controller  Failed to run phase 0 of action delete: command terminated with exit code 1

# Example of failure events of ActionSet emitted to its associated Blueprint:
$ kubectl --namespace kanister describe blueprint <Blueprint Name>
Events:
  Type     Reason                           Age   From                 Message
  ----     ------                           ----  ----                 -------
  Normal   Added                            4m   Kanister Controller  Added blueprint 'Blueprint Name'
  Warning  ActionSetFailed Action: delete   1m   Kanister Controller  Failed to run phase 0 of action delete: command terminated with exit code 1
```

If you ever need to debug a live Kanister system and the information
available in ActionSets you might have created is not enough, looking at
the Kanister controller logs might help. Assuming you have deployed the
controller in the `kanister` namespace, you can use the following
commands to get controller logs.

``` bash
$ kubectl get pods --namespace kanister
NAME                                           READY     STATUS    RESTARTS   AGE
release-kanister-operator-1484730505-l443d   1/1       Running   0          1m

$ kubectl logs -f <operator-pod-name-from-above> --namespace kanister
```

If you are not successful in verifying the reason behind the failure,
please reach out to us on
[Slack](https://join.slack.com/t/kanisterio/shared_invite/enQtNzg2MDc4NzA0ODY4LTU1NDU2NDZhYjk3YmE5MWNlZWMwYzk1NjNjOGQ3NjAyMjcxMTIyNTE1YzZlMzgwYmIwNWFkNjU0NGFlMzNjNTk)
or file an issue on
[GitHub](https://github.com/kanisterio/kanister/issues). A [mailing
list](https://groups.google.com/forum/#!forum/kanisterio) is also
available if needed.

## Validating webhook for Blueprints

For the validating webhook to work, the Kubernetes API Server needs to
connect to port `9443` of the Kanister operator. If your cluster has a
firewall setup, it has to be configured to allow that communication.

### GKE

If you get an error while applying a blueprint, that the webhook can\'t
be reached, check if your firewall misses a rule for port `9443`:

``` console
$ kubectl apply -f blueprint.yaml
Error from server (InternalError): error when creating "blueprint.yaml": Internal error occurred: failed calling webhook "blueprints.cr.kanister.io": failed to call webhook: Post "https://kanister-kanister-operator.kanister.svc:443/validate/v1alpha1/blueprint?timeout=5s": context deadline exceeded
```

See [GKE: Adding firewall rules for specific use
cases](https://cloud.google.com/kubernetes-engine/docs/how-to/private-clusters#add_firewall_rules)
and [kubernetes/kubernetes: Using non-443 ports for admission webhooks
requires firewall rule in
GKE](https://github.com/kubernetes/kubernetes/issues/79739) for more
details.
