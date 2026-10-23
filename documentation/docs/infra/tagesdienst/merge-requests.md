# Merge Requests

Location for topics merge requests: <https://gitlab.pnet.ch/activemq/instances/-/merge_requests>

Changes in the namespaces repository are merged automatically if possible. However, auto-merging can fail 
because of the following reasons. If this is the case, add yourself as reviewer, carefully check the change logs and 
reject or merge manually.

### MissingCertificate
**Reason:** The MR author is **defining** a new identity, of which no certificate in CES exists yet.

**Action:**  
The merge operator prompts the user to provide the target host(s) or namespace. If this information is missing, leave a 
comment asking the author to provide it. Once available, the certificate must be created in CES. Instructions can be 
found here: [create-certificate](/docs/default/resource/activemq/infra/guides/certificate/)

### MissingBlockStorageAnnotation
**Reason:** The block storage annotation on the definition of the target namespace is missing.

**Action:**
The namespace where Artemis will be deployed needs a block storage annotation. Instructions on how to set the annotation 
can be found in the [Getting Started guide](/docs/default/resource/activemq/getting-started/). If the user doesn't resolve this based on the MR 
message, you can share the link to the guide
directly.

### PropertiesChange
**Reason:** Custom properties are being set. Normally, the operator applies all necessary configuration based on the 
PFActiveMQ custom resource. However, in some cases—such as testing or temporary fixes—it may be necessary to override
Artemis properties manually, for example:

````yaml
    properties:
      - maxDiskUsage=50
      - addressSettings."mqp.#".addressFullMessagePolicy=PAGE
````
**Action:** Review the properties carefully and critically assess whether they are needed. If you're unsure, check if
similar settings exist in a lower environment and / or consult an Artemis specialist for a review.


### JavaArgsChange
**Reason:** Custom javaArgs are being set. Normally, the operator applies all necessary configuration based on the 
PFActiveMQ custom resource. However, in some cases—such as testing or temporary fixes—it may be necessary to override
Artemis javaArgs manually, for example:

````yaml
    javaArgs:
      - "-XX:MaxRAMPercentage=40"
````
**Action:** Review the javaArgs carefully and critically assess whether they are needed. If you're unsure, check if
similar settings exist in a lower environment and / or consult an Artemis specialist for a review.
