.. _troubleshooting:

Troubleshooting
***************

If an ActionSet fails to perform an action, then the failure events can be seen
in the respective ActionSet as well as its associated Blueprint by using the
following commands:

.. code-block:: bash

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

If you ever need to debug a live Kanister system and the information
available in ActionSets you might have created is not enough, looking
at the Kanister controller logs might help. Assuming you have deployed
the controller in the ``kanister`` namespace, you can use the following
commands to get controller logs.

.. code-block:: bash

  $ kubectl get pods --namespace kanister
  NAME                                           READY     STATUS    RESTARTS   AGE
  release-kanister-operator-1484730505-l443d   1/1       Running   0          1m

  $ kubectl logs -f <operator-pod-name-from-above> --namespace kanister


If you are not successful in verifying the reason behind the failure,
please reach out to us on `Slack
<https://join.slack.com/t/kanisterio/shared_invite/enQtNzg2MDc4NzA0ODY4LTU1NDU2NDZhYjk3YmE5MWNlZWMwYzk1NjNjOGQ3NjAyMjcxMTIyNTE1YzZlMzgwYmIwNWFkNjU0NGFlMzNjNTk>`_
or file an issue on `GitHub
<https://github.com/kanisterio/kanister/issues>`_. A `mailing list
<https://groups.google.com/forum/#!forum/kanisterio>`_ is also
available if needed.
