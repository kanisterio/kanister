<p>Packages:</p>
<ul>
<li>
<a href="#cr%2fv1alpha1">cr/v1alpha1</a>
</li>
</ul>
<h2 id="cr/v1alpha1">cr/v1alpha1</h2>
<p>
<p>Package v1alpha1 is the v1alpha1 version of the API.</p>
</p>
Resource Types:
<ul><li>
<a href="#cr/v1alpha1.ActionSet">ActionSet</a>
</li><li>
<a href="#cr/v1alpha1.Blueprint">Blueprint</a>
</li><li>
<a href="#cr/v1alpha1.Profile">Profile</a>
</li></ul>
<h3 id="cr/v1alpha1.ActionSet">ActionSet
</h3>
<p>
<p>ActionSet describes kanister actions.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>apiVersion</code><br/>
string</td>
<td>
<code>
cr/v1alpha1
</code>
</td>
</tr>
<tr>
<td>
<code>kind</code><br/>
string
</td>
<td><code>ActionSet</code></td>
</tr>
<tr>
<td>
<code>metadata</code><br/>
<em>
<a href="https://v1-18.docs.kubernetes.io/docs/reference/generated/kubernetes-api/v1.18/#objectmeta-v1-meta">
Kubernetes meta/v1.ObjectMeta
</a>
</em>
</td>
<td>
Refer to the Kubernetes API documentation for the fields of the
<code>metadata</code> field.
</td>
</tr>
<tr>
<td>
<code>spec</code><br/>
<em>
<a href="#cr/v1alpha1.ActionSetSpec">
ActionSetSpec
</a>
</em>
</td>
<td>
<p>Spec defines the specification for the actionset.
The specification includes a list of Actions to be performed. Each Action includes details
about the referenced Blueprint and other objects used to perform the defined action.</p>
<br/>
<br/>
<table>
</table>
</td>
</tr>
<tr>
<td>
<code>status</code><br/>
<em>
<a href="#cr/v1alpha1.ActionSetStatus">
ActionSetStatus
</a>
</em>
</td>
<td>
<p>Status refers to the current status of the Kanister actions.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="cr/v1alpha1.Blueprint">Blueprint
</h3>
<p>
<p>Blueprint describes kanister actions.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>apiVersion</code><br/>
string</td>
<td>
<code>
cr/v1alpha1
</code>
</td>
</tr>
<tr>
<td>
<code>kind</code><br/>
string
</td>
<td><code>Blueprint</code></td>
</tr>
<tr>
<td>
<code>metadata</code><br/>
<em>
<a href="https://v1-18.docs.kubernetes.io/docs/reference/generated/kubernetes-api/v1.18/#objectmeta-v1-meta">
Kubernetes meta/v1.ObjectMeta
</a>
</em>
</td>
<td>
Refer to the Kubernetes API documentation for the fields of the
<code>metadata</code> field.
</td>
</tr>
<tr>
<td>
<code>actions</code><br/>
<em>
<a href="#cr/v1alpha1.*./pkg/apis/cr/v1alpha1.BlueprintAction">
map[string]*./pkg/apis/cr/v1alpha1.BlueprintAction
</a>
</em>
</td>
<td>
<p>Actions is the list of actions constructing the Blueprint.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="cr/v1alpha1.Profile">Profile
</h3>
<p>
<p>Profile captures information about a storage location for backup artifacts and
corresponding credentials, that will be made available to a Blueprint phase.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>apiVersion</code><br/>
string</td>
<td>
<code>
cr/v1alpha1
</code>
</td>
</tr>
<tr>
<td>
<code>kind</code><br/>
string
</td>
<td><code>Profile</code></td>
</tr>
<tr>
<td>
<code>metadata</code><br/>
<em>
<a href="https://v1-18.docs.kubernetes.io/docs/reference/generated/kubernetes-api/v1.18/#objectmeta-v1-meta">
Kubernetes meta/v1.ObjectMeta
</a>
</em>
</td>
<td>
Refer to the Kubernetes API documentation for the fields of the
<code>metadata</code> field.
</td>
</tr>
<tr>
<td>
<code>location</code><br/>
<em>
<a href="#cr/v1alpha1.Location">
Location
</a>
</em>
</td>
<td>
<p>Location provides the information about the object storage that is going to be used by Kanister to upload the backup objects.</p>
</td>
</tr>
<tr>
<td>
<code>credential</code><br/>
<em>
<a href="#cr/v1alpha1.Credential">
Credential
</a>
</em>
</td>
<td>
<p>Credential represents the credentials associated with the Location.</p>
</td>
</tr>
<tr>
<td>
<code>skipSSLVerify</code><br/>
<em>
bool
</em>
</td>
<td>
<p>SkipSSLVerify is a boolean that specifies whether skipping SSL verification
is allowed when operating with the Location.
If omitted from the CR definition, it defaults to false</p>
</td>
</tr>
</tbody>
</table>
<h3 id="cr/v1alpha1.ActionProgress">ActionProgress
</h3>
<p>
(<em>Appears on:</em>
<a href="#cr/v1alpha1.ActionSetStatus">ActionSetStatus</a>)
</p>
<p>
<p>ActionProgress provides information on the progress of an action.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>runningPhase</code><br/>
<em>
string
</em>
</td>
<td>
<p>RunningPhase represents which phase of the action is being run</p>
</td>
</tr>
<tr>
<td>
<code>percentCompleted</code><br/>
<em>
string
</em>
</td>
<td>
<p>PercentCompleted is computed by assessing the number of completed phases
against the the total number of phases.</p>
</td>
</tr>
<tr>
<td>
<code>lastTransitionTime</code><br/>
<em>
<a href="https://v1-18.docs.kubernetes.io/docs/reference/generated/kubernetes-api/v1.18/#time-v1-meta">
Kubernetes meta/v1.Time
</a>
</em>
</td>
<td>
<p>LastTransitionTime represents the last date time when the progress status
was received.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="cr/v1alpha1.ActionSetSpec">ActionSetSpec
</h3>
<p>
(<em>Appears on:</em>
<a href="#cr/v1alpha1.ActionSet">ActionSet</a>)
</p>
<p>
<p>ActionSetSpec is the specification for the actionset.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>actions</code><br/>
<em>
<a href="#cr/v1alpha1.ActionSpec">
[]ActionSpec
</a>
</em>
</td>
<td>
<p>Actions represents a list of Actions that need to be performed by the actionset.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="cr/v1alpha1.ActionSetStatus">ActionSetStatus
</h3>
<p>
(<em>Appears on:</em>
<a href="#cr/v1alpha1.ActionSet">ActionSet</a>)
</p>
<p>
<p>ActionSetStatus is the status for the actionset. This should only be updated by the controller.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>state</code><br/>
<em>
<a href="#cr/v1alpha1.State">
State
</a>
</em>
</td>
<td>
<p>State represents the current state of the actionset.
There are four possible values: &ldquo;Pending&rdquo;, &ldquo;Running&rdquo;, &ldquo;Failed&rdquo;, and &ldquo;Complete&rdquo;.</p>
</td>
</tr>
<tr>
<td>
<code>actions</code><br/>
<em>
<a href="#cr/v1alpha1.ActionStatus">
[]ActionStatus
</a>
</em>
</td>
<td>
<p>Actions list represents the latest available observations of the current state of all the actions.</p>
</td>
</tr>
<tr>
<td>
<code>error</code><br/>
<em>
<a href="#cr/v1alpha1.Error">
Error
</a>
</em>
</td>
<td>
<p>Error contains the detailed error message of an actionset failure.</p>
</td>
</tr>
<tr>
<td>
<code>progress</code><br/>
<em>
<a href="#cr/v1alpha1.ActionProgress">
ActionProgress
</a>
</em>
</td>
<td>
<p>Progress provides information on the progress of a running actionset.
This includes the percentage of completion of an actionset and the phase that is
currently being executed.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="cr/v1alpha1.ActionSpec">ActionSpec
</h3>
<p>
(<em>Appears on:</em>
<a href="#cr/v1alpha1.ActionSetSpec">ActionSetSpec</a>)
</p>
<p>
<p>ActionSpec is the specification for a single Action.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>name</code><br/>
<em>
string
</em>
</td>
<td>
<p>Name is the action we&rsquo;ll perform. For example: <code>backup</code> or <code>restore</code>.</p>
</td>
</tr>
<tr>
<td>
<code>object</code><br/>
<em>
<a href="#cr/v1alpha1.ObjectReference">
ObjectReference
</a>
</em>
</td>
<td>
<p>Object refers to the thing we&rsquo;ll perform this action on.</p>
</td>
</tr>
<tr>
<td>
<code>blueprint</code><br/>
<em>
string
</em>
</td>
<td>
<p>Blueprint with instructions on how to execute this action.</p>
</td>
</tr>
<tr>
<td>
<code>artifacts</code><br/>
<em>
<a href="#cr/v1alpha1.Artifact">
map[string]./pkg/apis/cr/v1alpha1.Artifact
</a>
</em>
</td>
<td>
<p>Artifacts will be passed as inputs into this phase.</p>
</td>
</tr>
<tr>
<td>
<code>configMaps</code><br/>
<em>
<a href="#cr/v1alpha1.ObjectReference">
map[string]./pkg/apis/cr/v1alpha1.ObjectReference
</a>
</em>
</td>
<td>
<p>ConfigMaps that we&rsquo;ll get and pass into the blueprint.</p>
</td>
</tr>
<tr>
<td>
<code>secrets</code><br/>
<em>
<a href="#cr/v1alpha1.ObjectReference">
map[string]./pkg/apis/cr/v1alpha1.ObjectReference
</a>
</em>
</td>
<td>
<p>Secrets that we&rsquo;ll get and pass into the blueprint.</p>
</td>
</tr>
<tr>
<td>
<code>profile</code><br/>
<em>
<a href="#cr/v1alpha1.ObjectReference">
ObjectReference
</a>
</em>
</td>
<td>
<p>Profile is use to specify the location where store artifacts and the
credentials authorized to access them.</p>
</td>
</tr>
<tr>
<td>
<code>podOverride</code><br/>
<em>
<a href="#cr/v1alpha1.JSONMap">
JSONMap
</a>
</em>
</td>
<td>
<p>PodOverride is used to specify pod specs that will override the
default pod specs</p>
</td>
</tr>
<tr>
<td>
<code>options</code><br/>
<em>
map[string]string
</em>
</td>
<td>
<p>Options will be used to specify additional values
to be used in the Blueprint.</p>
</td>
</tr>
<tr>
<td>
<code>preferredVersion</code><br/>
<em>
string
</em>
</td>
<td>
<p>PreferredVersion will be used to select the preferred version of Kanister functions
to be executed for this action</p>
</td>
</tr>
</tbody>
</table>
<h3 id="cr/v1alpha1.ActionStatus">ActionStatus
</h3>
<p>
(<em>Appears on:</em>
<a href="#cr/v1alpha1.ActionSetStatus">ActionSetStatus</a>)
</p>
<p>
<p>ActionStatus is updated as we execute phases.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>name</code><br/>
<em>
string
</em>
</td>
<td>
<p>Name is the action we&rsquo;ll perform. For example: <code>backup</code> or <code>restore</code>.</p>
</td>
</tr>
<tr>
<td>
<code>object</code><br/>
<em>
<a href="#cr/v1alpha1.ObjectReference">
ObjectReference
</a>
</em>
</td>
<td>
<p>Object refers to the thing we&rsquo;ll perform this action on.</p>
</td>
</tr>
<tr>
<td>
<code>blueprint</code><br/>
<em>
string
</em>
</td>
<td>
<p>Blueprint with instructions on how to execute this action.</p>
</td>
</tr>
<tr>
<td>
<code>phases</code><br/>
<em>
<a href="#cr/v1alpha1.Phase">
[]Phase
</a>
</em>
</td>
<td>
<p>Phases are sub-actions an are executed sequentially.</p>
</td>
</tr>
<tr>
<td>
<code>artifacts</code><br/>
<em>
<a href="#cr/v1alpha1.Artifact">
map[string]./pkg/apis/cr/v1alpha1.Artifact
</a>
</em>
</td>
<td>
<p>Artifacts created by this phase.</p>
</td>
</tr>
<tr>
<td>
<code>deferPhase</code><br/>
<em>
<a href="#cr/v1alpha1.Phase">
Phase
</a>
</em>
</td>
<td>
<p>DeferPhase is the phase that is executed at the end of an action
irrespective of the status of other phases in the action</p>
</td>
</tr>
</tbody>
</table>
<h3 id="cr/v1alpha1.Artifact">Artifact
</h3>
<p>
(<em>Appears on:</em>
<a href="#cr/v1alpha1.ActionSpec">ActionSpec</a>, 
<a href="#cr/v1alpha1.ActionStatus">ActionStatus</a>, 
<a href="#cr/v1alpha1.BlueprintAction">BlueprintAction</a>)
</p>
<p>
<p>Artifact tracks objects produced by an action.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>keyValue</code><br/>
<em>
map[string]string
</em>
</td>
<td>
<p>KeyValue represents key-value pair artifacts produced by the action.</p>
</td>
</tr>
<tr>
<td>
<code>kopiaSnapshot</code><br/>
<em>
string
</em>
</td>
<td>
<p>KopiaSnapshot captures the kopia snapshot information
produced as a JSON string by kando command in phases of an action.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="cr/v1alpha1.BlueprintAction">BlueprintAction
</h3>
<p>
<p>BlueprintAction describes the set of phases that constitute an action.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>name</code><br/>
<em>
string
</em>
</td>
<td>
<p>Name contains the name of the action.</p>
</td>
</tr>
<tr>
<td>
<code>kind</code><br/>
<em>
string
</em>
</td>
<td>
<p>Kind contains the resource on which this action has to be performed.</p>
</td>
</tr>
<tr>
<td>
<code>configMapNames</code><br/>
<em>
[]string
</em>
</td>
<td>
<p>ConfigMapNames is used to specify the config map names that can be used later in the action phases.</p>
</td>
</tr>
<tr>
<td>
<code>secretNames</code><br/>
<em>
[]string
</em>
</td>
<td>
<p>List of Kubernetes secret names used in action phases.</p>
</td>
</tr>
<tr>
<td>
<code>inputArtifactNames</code><br/>
<em>
[]string
</em>
</td>
<td>
<p>InputArtifactNames is the list of Artifact names that were set from previous action and can be consumed in the current action.</p>
</td>
</tr>
<tr>
<td>
<code>outputArtifacts</code><br/>
<em>
<a href="#cr/v1alpha1.Artifact">
map[string]./pkg/apis/cr/v1alpha1.Artifact
</a>
</em>
</td>
<td>
<p>OutputArtifacts is the map of rendered artifacts produced by the BlueprintAction.</p>
</td>
</tr>
<tr>
<td>
<code>phases</code><br/>
<em>
<a href="#cr/v1alpha1.BlueprintPhase">
[]BlueprintPhase
</a>
</em>
</td>
<td>
<p>Phases is the list of BlueprintPhases which are invoked in order when executing this action.</p>
</td>
</tr>
<tr>
<td>
<code>deferPhase</code><br/>
<em>
<a href="#cr/v1alpha1.BlueprintPhase">
BlueprintPhase
</a>
</em>
</td>
<td>
<p>DeferPhase is invoked after the execution of Phases that are defined for an action.
A DeferPhase is executed regardless of the statuses of the other phases of the action.
A DeferPhase can be used for cleanup operations at the end of an action.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="cr/v1alpha1.BlueprintPhase">BlueprintPhase
</h3>
<p>
(<em>Appears on:</em>
<a href="#cr/v1alpha1.BlueprintAction">BlueprintAction</a>)
</p>
<p>
<p>BlueprintPhase is a an individual unit of execution.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>func</code><br/>
<em>
string
</em>
</td>
<td>
<p>Func is the name of a registered Kanister function.</p>
</td>
</tr>
<tr>
<td>
<code>name</code><br/>
<em>
string
</em>
</td>
<td>
<p>Name contains name of the phase.</p>
</td>
</tr>
<tr>
<td>
<code>objects</code><br/>
<em>
<a href="#cr/v1alpha1.ObjectReference">
map[string]./pkg/apis/cr/v1alpha1.ObjectReference
</a>
</em>
</td>
<td>
<p>ObjectRefs represents a map of references to the Kubernetes objects that
can later be used in the <code>Args</code> of the function.</p>
</td>
</tr>
<tr>
<td>
<code>args</code><br/>
<em>
map[string]interface{}
</em>
</td>
<td>
<p>Args represents a map of named arguments that the controller will pass to the Kanister function.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="cr/v1alpha1.Credential">Credential
</h3>
<p>
(<em>Appears on:</em>
<a href="#cr/v1alpha1.Profile">Profile</a>)
</p>
<p>
<p>Credential</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>type</code><br/>
<em>
<a href="#cr/v1alpha1.CredentialType">
CredentialType
</a>
</em>
</td>
<td>
<p>Type represents the information about how the credentials are provided for the respective object storage.</p>
</td>
</tr>
<tr>
<td>
<code>keyPair</code><br/>
<em>
<a href="#cr/v1alpha1.KeyPair">
KeyPair
</a>
</em>
</td>
<td>
<p>KeyPair represents the key-value map used for the Credential of Type KeyPair.</p>
</td>
</tr>
<tr>
<td>
<code>secret</code><br/>
<em>
<a href="#cr/v1alpha1.ObjectReference">
ObjectReference
</a>
</em>
</td>
<td>
<p>Secret represents the Kubernetes Secret Object used for the Credential of Type Secret.</p>
</td>
</tr>
<tr>
<td>
<code>kopiaServerSecret</code><br/>
<em>
<a href="#cr/v1alpha1.KopiaServerSecret">
KopiaServerSecret
</a>
</em>
</td>
<td>
<p>KopiaServerSecret represents the secret being used by Credential of Type Kopia.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="cr/v1alpha1.CredentialType">CredentialType
(<code>string</code> alias)</p></h3>
<p>
(<em>Appears on:</em>
<a href="#cr/v1alpha1.Credential">Credential</a>)
</p>
<p>
<p>CredentialType</p>
</p>
<table>
<thead>
<tr>
<th>Value</th>
<th>Description</th>
</tr>
</thead>
<tbody><tr><td><p>&#34;keyPair&#34;</p></td>
<td></td>
</tr><tr><td><p>&#34;kopia&#34;</p></td>
<td></td>
</tr><tr><td><p>&#34;secret&#34;</p></td>
<td></td>
</tr></tbody>
</table>
<h3 id="cr/v1alpha1.Error">Error
</h3>
<p>
(<em>Appears on:</em>
<a href="#cr/v1alpha1.ActionSetStatus">ActionSetStatus</a>)
</p>
<p>
<p>Error represents an error that occurred when executing an actionset.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>message</code><br/>
<em>
string
</em>
</td>
<td>
<p>Message is the actual error message that is displayed in case of errors.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="cr/v1alpha1.JSONMap">JSONMap
(<code>map[string]interface{}</code> alias)</p></h3>
<p>
(<em>Appears on:</em>
<a href="#cr/v1alpha1.ActionSpec">ActionSpec</a>)
</p>
<p>
<p>JSONMap contains PodOverride specs.</p>
</p>
<h3 id="cr/v1alpha1.KeyPair">KeyPair
</h3>
<p>
(<em>Appears on:</em>
<a href="#cr/v1alpha1.Credential">Credential</a>)
</p>
<p>
<p>KeyPair</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>idField</code><br/>
<em>
string
</em>
</td>
<td>
<p>IDField specifies the corresponding key in the secret where the AWS Key ID value is stored.</p>
</td>
</tr>
<tr>
<td>
<code>secretField</code><br/>
<em>
string
</em>
</td>
<td>
<p>SecretField specifies the corresponding key in the secret where the AWS Secret Key value is stored.</p>
</td>
</tr>
<tr>
<td>
<code>secret</code><br/>
<em>
<a href="#cr/v1alpha1.ObjectReference">
ObjectReference
</a>
</em>
</td>
<td>
<p>Secret represents a Kubernetes Secret object storing the KeyPair credentials.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="cr/v1alpha1.KopiaServerSecret">KopiaServerSecret
</h3>
<p>
(<em>Appears on:</em>
<a href="#cr/v1alpha1.Credential">Credential</a>)
</p>
<p>
<p>KopiaServerSecret contains credentials to connect to Kopia server</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>username</code><br/>
<em>
string
</em>
</td>
<td>
<p>Username represents the username used to connect to the Kopia Server.</p>
</td>
</tr>
<tr>
<td>
<code>hostname</code><br/>
<em>
string
</em>
</td>
<td>
<p>Hostname represents the hostname used to connect to the Kopia Server.</p>
</td>
</tr>
<tr>
<td>
<code>userPassphrase</code><br/>
<em>
<a href="#cr/v1alpha1.KopiaServerSecretRef">
KopiaServerSecretRef
</a>
</em>
</td>
<td>
<p>UserPassphrase is the user password used to connect to the Kopia Server.</p>
</td>
</tr>
<tr>
<td>
<code>tlsCert</code><br/>
<em>
<a href="#cr/v1alpha1.KopiaServerSecretRef">
KopiaServerSecretRef
</a>
</em>
</td>
<td>
<p>TLSCert is the certificate used to connect to the Kopia Server.</p>
</td>
</tr>
<tr>
<td>
<code>connectOptions</code><br/>
<em>
map[string]int
</em>
</td>
<td>
<p>ConnectOptions represents a map of options which can be used to connect to the Kopia Server.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="cr/v1alpha1.KopiaServerSecretRef">KopiaServerSecretRef
</h3>
<p>
(<em>Appears on:</em>
<a href="#cr/v1alpha1.KopiaServerSecret">KopiaServerSecret</a>)
</p>
<p>
<p>KopiaServerSecretRef refers to K8s secrets containing Kopia creds</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>key</code><br/>
<em>
string
</em>
</td>
<td>
<p>Key represents the corresponding key in the secret where the required
credential or certificate value is stored.</p>
</td>
</tr>
<tr>
<td>
<code>secret</code><br/>
<em>
<a href="#cr/v1alpha1.ObjectReference">
ObjectReference
</a>
</em>
</td>
<td>
<p>Secret is the K8s secret object where the creds related to the Kopia Server are stored.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="cr/v1alpha1.Location">Location
</h3>
<p>
(<em>Appears on:</em>
<a href="#cr/v1alpha1.Profile">Profile</a>)
</p>
<p>
<p>Location</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>type</code><br/>
<em>
<a href="#cr/v1alpha1.LocationType">
LocationType
</a>
</em>
</td>
<td>
<p>Type specifies the kind of object storage that would be used to upload the
backup objects. Currently supported values are: &ldquo;GCS&rdquo;, &ldquo;S3Compliant&rdquo;,
and &ldquo;Azure&rdquo;.</p>
</td>
</tr>
<tr>
<td>
<code>bucket</code><br/>
<em>
string
</em>
</td>
<td>
<p>Bucket represents the bucket on the object storage where the backup is uploaded.</p>
</td>
</tr>
<tr>
<td>
<code>endpoint</code><br/>
<em>
string
</em>
</td>
<td>
<p>Endpoint specifies the endpoint where the object storage is accessible at.</p>
</td>
</tr>
<tr>
<td>
<code>prefix</code><br/>
<em>
string
</em>
</td>
<td>
<p>Prefix is the string that would be prepended to the object path in the
bucket where the backup objects are uploaded.</p>
</td>
</tr>
<tr>
<td>
<code>region</code><br/>
<em>
string
</em>
</td>
<td>
<p>Region represents the region of the bucket specified above.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="cr/v1alpha1.LocationType">LocationType
(<code>string</code> alias)</p></h3>
<p>
(<em>Appears on:</em>
<a href="#cr/v1alpha1.Location">Location</a>)
</p>
<p>
<p>LocationType</p>
</p>
<table>
<thead>
<tr>
<th>Value</th>
<th>Description</th>
</tr>
</thead>
<tbody><tr><td><p>&#34;azure&#34;</p></td>
<td></td>
</tr><tr><td><p>&#34;gcs&#34;</p></td>
<td></td>
</tr><tr><td><p>&#34;kopia&#34;</p></td>
<td></td>
</tr><tr><td><p>&#34;s3Compliant&#34;</p></td>
<td></td>
</tr></tbody>
</table>
<h3 id="cr/v1alpha1.ObjectReference">ObjectReference
</h3>
<p>
(<em>Appears on:</em>
<a href="#cr/v1alpha1.ActionSpec">ActionSpec</a>, 
<a href="#cr/v1alpha1.ActionStatus">ActionStatus</a>, 
<a href="#cr/v1alpha1.BlueprintPhase">BlueprintPhase</a>, 
<a href="#cr/v1alpha1.Credential">Credential</a>, 
<a href="#cr/v1alpha1.KeyPair">KeyPair</a>, 
<a href="#cr/v1alpha1.KopiaServerSecretRef">KopiaServerSecretRef</a>)
</p>
<p>
<p>ObjectReference refers to a kubernetes object.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>apiVersion</code><br/>
<em>
string
</em>
</td>
<td>
<p>API version of the referent.</p>
</td>
</tr>
<tr>
<td>
<code>group</code><br/>
<em>
string
</em>
</td>
<td>
<p>API Group of the referent.</p>
</td>
</tr>
<tr>
<td>
<code>resource</code><br/>
<em>
string
</em>
</td>
<td>
<p>Resource name of the referent.</p>
</td>
</tr>
<tr>
<td>
<code>kind</code><br/>
<em>
string
</em>
</td>
<td>
<p>Kind of the referent.
More info: <a href="https://git.k8s.io/community/contributors/devel/api-conventions.md#types-kinds">https://git.k8s.io/community/contributors/devel/api-conventions.md#types-kinds</a></p>
</td>
</tr>
<tr>
<td>
<code>name</code><br/>
<em>
string
</em>
</td>
<td>
<p>Name of the referent.
More info: <a href="http://kubernetes.io/docs/user-guide/identifiers#names">http://kubernetes.io/docs/user-guide/identifiers#names</a></p>
</td>
</tr>
<tr>
<td>
<code>namespace</code><br/>
<em>
string
</em>
</td>
<td>
<p>Namespace of the referent.
More info: <a href="http://kubernetes.io/docs/user-guide/namespaces">http://kubernetes.io/docs/user-guide/namespaces</a></p>
</td>
</tr>
</tbody>
</table>
<h3 id="cr/v1alpha1.Phase">Phase
</h3>
<p>
(<em>Appears on:</em>
<a href="#cr/v1alpha1.ActionStatus">ActionStatus</a>)
</p>
<p>
<p>Phase is subcomponent of an action.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>name</code><br/>
<em>
string
</em>
</td>
<td>
<p>Name represents the name of the Blueprint phase.</p>
</td>
</tr>
<tr>
<td>
<code>state</code><br/>
<em>
<a href="#cr/v1alpha1.State">
State
</a>
</em>
</td>
<td>
<p>State represents the current state of execution of the Blueprint phase.</p>
</td>
</tr>
<tr>
<td>
<code>output</code><br/>
<em>
map[string]interface{}
</em>
</td>
<td>
<p>Output is the map of output artifacts produced by the Blueprint phase.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="cr/v1alpha1.State">State
(<code>string</code> alias)</p></h3>
<p>
(<em>Appears on:</em>
<a href="#cr/v1alpha1.ActionSetStatus">ActionSetStatus</a>, 
<a href="#cr/v1alpha1.Phase">Phase</a>)
</p>
<p>
<p>State is the current state of a phase of execution.</p>
</p>
<table>
<thead>
<tr>
<th>Value</th>
<th>Description</th>
</tr>
</thead>
<tbody><tr><td><p>&#34;complete&#34;</p></td>
<td><p>StateComplete means this action or phase finished successfully.</p>
</td>
</tr><tr><td><p>&#34;failed&#34;</p></td>
<td><p>StateFailed means this action or phase was unsuccessful.</p>
</td>
</tr><tr><td><p>&#34;pending&#34;</p></td>
<td><p>StatePending mean this action or phase has yet to be executed.</p>
</td>
</tr><tr><td><p>&#34;running&#34;</p></td>
<td><p>StateRunning means this action or phase is currently executing.</p>
</td>
</tr></tbody>
</table>
<hr/>
<p><em>
Generated with <code>gen-crd-api-reference-docs</code>
.
</em></p>
