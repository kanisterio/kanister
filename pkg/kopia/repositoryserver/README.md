To generate the RepositoryServer CRD: (Ref.: https://www.youtube.com/watch?v=89PdRvRUcPU&t=640s)

1. Created a project separately

2. Used the above CR manifest fields to create the client-go type of our CR

3. Installed the code-generator library to auto generate the clientset, listers and informers
   ╰─$ go get k8s.io/code-generator

4. Auto-generated code using following command for code-generator
   ╰─$ ~/go/src/k8s.io/code-generator/generate-groups.sh all github.com/kanisterio/kanister/pkg/kopia/repositoryserver/pkg/client github.com/shlokchaudhari9/kopiareposerver/pkg/apis cr.kanister.io:v1alpha1 --go-header-file ~/go/src/k8s.io/code-generator/hack/boilerplate.go.txt

5. Ran this command to install the controller-gen stable release from the kubebuilder contoller-tools
   ╰─$ go get sigs.k8s.io/controller-tools/cmd/controller-gen@v0.4.1

6. Created CRD manifest using following command
   ╰─$ controller-gen paths=github.com/kanisterio/kanister/pkg/kopia/repositoryserver/pkg/apis/cr.kanister.io/v1alpha1/  crd:trivialVersions=true  crd:crdVersions=v1  output:crd:artifacts:config=manifests