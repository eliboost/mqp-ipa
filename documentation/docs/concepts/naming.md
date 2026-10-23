# Naming

Guideline for naming things in ActiveMQ Artemis.

## Instances

Full-Format: `<context>-mqp-<application>[-<sub-application>]-<cluster-identifier>` for example: `e1-mqp-esbpf-a`

**context**

* **Format:** `e1|t1|t2|u1|p1`
* **Examples:** e1, t1, t2, p1
* **Description:** context of the instance. This should be the environment of the instance, i.e. on an e1
k8s-cluster only e1 contexts should be present. However, on t1 k8s-nucleus cluster only t1 and t2 are
allowed.

**application**

* **Format:** `[a-z0-9-]+`
* **Examples:** bancs, dcs, salmon
* **Description:** Short application name. Use the application abbreviation
  from ITAM, in lower case.

**sub-application**

* **Format:** `[a-z0-9-]+`
* **Examples:** bpay, falcon, pp
* **Description:** This is optional. Short sub-application, application module name from ITAM if available.

**cluster-identifier**

* **Format:** `[a-z]+`
* **Examples:** a, b, c
* **Description:** If you need multiple instances, you can increase this identifier.

## Queues

Full-Format: `<resource-type>.<application>.[<identifier>_]<subject>` for example: `mqp.esbpf.Queue_1_Test`

**resource-type**

* **Format:** `mqp`
* **Description:** mqp: Identifier for the platform (MQP) the queue is related to.

**application**

* **Format:** `[a-z0-9-]+`
* **Examples:** esbpf, egw, bpmp
* **Description:** short application name. Use the application abbreviation from ITAM, in lower case.

**subject**

* **Format:** `[a-zA-Z0-9]+(_[a-zA-Z0-9]+)*`
* **Examples:** mqp.csgw.In_SMS_FAST_Queue , mqp.bancs.PF_15C_SWPAINEXTI,
  mqp.egw.In_Queue_ECM_Trigger
* **Description:** Describes what the queue is transporting or what the queue is used for. If you
  want to structure several elements to form your subject, separate them with `_` (underscore).

**identifier**

* **Format:** `In_Queue, Out_Queue`
* **Examples:** mqp.egw.In_Queue_ECM_Trigger
* **Description:** This is optional. The identifier indicates that it refers to an external queue, which is used for 
communication across application scopes via ESBPF, and is reserved for such use cases.

## Identity

Full-Format: `<prefix>-<resource-type>-<application>[-<sub-application>]-<context>` for example: `a-pf-identity-mqp-bancs-bpay-e1`

**prefix**

* **Format:** `a-pf-identity`
* **Description:**  a-pf-identity: Identifier prefix to indicate, what Certificate type it is.

**resource-type**

* **Format:** `mqp`
* **Description:**  mqp: Identifier for the platform (MQP) the identity is related to.

**application**

* **Format:** `[a-z0-9-]+`
* **Examples:** esbpf, egw, bpmp
* **Description:** Short application name. Use the application abbreviation from ITAM, in lower case.

**sub-application**

* **Format:** `[a-z0-9-]+`
* **Examples:** bpay, falcon, pp
* **Description:** This is optional. Short sub-application, application module name from ITAM if available.

**context**

* **Format:** `e1|t1|t2|u1|p1`
* **Examples:** e1, t1, t2, p1
* **Description:** context of the instance. This should be the environment of the instance, i.e. on an e1
  k8s-cluster only e1 contexts should be present. However, on t1 k8s-nucleus cluster only t1 and t2 are
  allowed.
