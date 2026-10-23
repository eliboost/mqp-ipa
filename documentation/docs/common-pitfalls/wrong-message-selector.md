# Wrong message-selector

A message in `mqp.dirmrm.In_Queue` was not consumed due to a mismatch between the expected JMS selector and the actual 
message header value. The configured `@JmsListener` was only matching `receiver = 'DIRM'`, but the actual value was
`DIRMRM`.

### Analysis Steps
The queue was manually paused to avoid message processing during investigation.

- Open the [MQP OPS UI](https://pfportal-test.pnet.ch/mqp-ops-ui/instances/instance-detail;name=t1-mqp-dirm-rm-a)  
- Navigate to queue: `mqp.dirmrm.In_Queue`, Replicat 1  
- Inspect the message details → Header_receiver = `DIRMRM`  
- Confirm no consumers processed the message

### Solution
The correct message-selector was set:
**Before:**

```java
@JmsListener(destination = "${apmpf.jms.app-in.queue}",
    containerFactory = "appJmsListenerFactory",
    selector = JmsConstants.HEADER_PROPERTY_RECEIVER + " = 'DIRM'"
            + " AND " + JmsConst
            ants.HEADER_PROPERTY_SENDER + " = '" + SOURCE_SYSTEM + "'",
    concurrency = "1")
```

**After:**

```java
@JmsListener(destination = "${apmpf.jms.app-in.queue}",
    containerFactory = "appJmsListenerFactory",
    selector = JmsConstants.HEADER_PROPERTY_RECEIVER + " = 'DIRMRM'"
            + " AND " + JmsConstants.HEADER_PROPERTY_SENDER + " = '" + SOURCE_SYSTEM + "'",
    concurrency = "1")
```
