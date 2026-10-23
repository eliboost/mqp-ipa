# Known Exceptions

## Connection Exceptions

#### `org.apache.activemq.artemis.api.core.ActiveMQConnectionTimedOutException: AMQ219013: Timed out waiting to receive cluster topology. Group:null`

**Library**  
`org.apache.activemq.artemis.jms.client.ActiveMQConnectionFactory`

**Description**  
This error occurs when the client times out while waiting for a response from the broker cluster. It's often related to 
cluster discovery or connectivity issues, especially in HA or clustered environments.

| Possible Cause                                                                                 | Solution                                                                                                         |
|------------------------------------------------------------------------------------------------|------------------------------------------------------------------------------------------------------------------|
| Broker is running in cluster mode, but client uses broker url without ha config                | Verify that your client uses a broker url with the necessary config `ha=false&useTopologyForLoadBalancing=false` |

---