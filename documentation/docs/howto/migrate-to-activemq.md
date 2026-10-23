# Migrate to ActiveMQ

A brief description on how to migrate from WLS JMS to ActiveMQ Artemis.

## Defining the IDM role for access
To be able to access the queues from your application, you need to configure the correct IDM role.
Refer to: [Getting Started](https://activemq.pnet.ch/docs/getting-started/)

## Create ActiveMQ instance using Self-Service
In this step you are configuring the new dedicated ActiveMQ instance for your application, containing all necessary queues.
Refer to [Use Self-Service](https://activemq.pnet.ch/docs/howto/use-self-service/)

## Make changes to JMS client configuration
You have to change the JNDI Initial Factory & Provider URL, as well as add some ActiveMQ specific SSL parameters.
Refer to [Client](https://activemq.pnet.ch/docs/howto/configuration/client/)

## Example configurations
In the following section you will find example configuration of other applications that have already completed the migration.
Refer to [Examples](https://activemq.pnet.ch/docs/examples)

## Recommended tests after migration
This section describes the recommended post-migration tests.
Refer to [Post-migration Tests](https://activemq.pnet.ch/docs/howto/post-migration-tests)