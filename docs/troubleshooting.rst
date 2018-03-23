.. _troubleshooting:

Troubleshooting
***************

If you ever need to debug a live Kanister system and the information
available in ActionSets you might have created is not enough, looking
at the Kanister controller logs might help. Assuming you have deployed
the controller in the `Kanister` namespace, you can use the following
commands to get controller logs.

.. code-block:: bash

  $ kubectl get pods --namespace kanister
  NAME                                           READY     STATUS    RESTARTS   AGE
  release-kanister-operator-1484730505-l443d   1/1       Running   0          1m

  $ kubectl logs -f <operator-pod-name-from-above> --namespace kanister


If you are not successful in verifying the reason behind the failure,
please reach out to us on `Slack
<https://kasten.typeform.com/to/QBcw8T>`_ or file an issue on `GitHub
<https://github.com/kanisterio/kanister/issues>`_. A `mailing list
<https://groups.google.com/forum/#!forum/kanisterio>`_ is also
available if needed.
