Configuring logs for specific ActionSets
----------------------------------------

Kanister uses structured logging to ensure that its logs can be easily
categorized, indexed and searched by downstream log aggregation software.

Extra fields can be added to the logs related to a specific ActionSet by adding
a label in the ActionSet with ``kanister.io`` prefix.

For example:

.. code-block:: yaml
  :linenos:

  apiVersion: cr.kanister.io/v1alpha1
  kind: ActionSet
  metadata:
    namespace: kanister
    name: myActionSet
    labels:
        kanister.io/myFieldName: myFieldValue

All logs concerning this ActionSet execution will have
``myFieldName`` field with ``myFieldValue`` value.



