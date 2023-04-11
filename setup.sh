make start-kind

make install-csi-hostpath-driver

make install-minio

helm install kanister ./helm/kanister-operator \
--namespace kanister \
--set image.repository=r4rajat/controller \
--set image.tag=v69 \
--set repositoryServerControllerImage.registry=r4rajat \
--set repositoryServerControllerImage.name=repo-server-controller \
--set repositoryServerControllerImage.tag=v69 \
--set controller.parallelism=10 \
--create-namespace

kubectl create -f pkg/customresource/repositoryserver.yaml -n kanister