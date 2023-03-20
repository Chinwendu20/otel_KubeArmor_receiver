### Overview

This receiver is created to fulfill the purpose of [adding opentelemetry support to Kubearmor](https://github.com/kubearmor/KubeArmor/issues/894). This receiver would convert the existing logs in opentelemetry to the [plog.logs format](https://github.com/open-telemetry/opentelemetry-collector/tree/main/pdata), this is the pipeline format in which logs are transported in memory in the collector.

[Stanza](https://github.com/observIQ/stanza) is a fast and lightweight log transport and processing agent. It has been donated to opentelemetry. Log based components in the collector contrib ( arepository for a repository for OpenTelemetry Collector components) use stanza as an intermediary to transform logs to plog.logs. Examples include:

- [filelogreceiver](https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/receiver/filelogreceiver)
- [syslogreceiver](https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/receiver/syslogreceiver)
- [tcplogreceiver](https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/receiver/tcplogreceiver)
- [udplogreceiver](https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/receiver/udplogreceiver)

I would be leveraging the same approach in creating the receiver.

To support Kubearmor, I would be creating the kubearmorlog (we could come up with a better name later) stanza input operator. An operator in Stanza is a task that helps us read from a file, parse the log, filter it and then push it to another log stream pipeline (similarly to the forwarding plugin of FluentD or Fluent Bit) and directly to the observability backend of your choice.

Similarly to the other agents on the market, there are several types of operators:

- Input

- Parser

- Transform

- Output (Link to source: [isitobservable.com](https://isitobservable.io/open-telemetry/what-is-stanza-and-what-does-it-do))

The stanza adapter in the [opentelemetry contrib](https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/pkg/stanza/adapter) takes care of intergrating stanza which does the conversion of logs to plog.logs format. To create the file input operator that would work with this, work has to be done to implement the stanza.LogReceiverType.


#### Proposed kubearmor receiver config

All opentelemetry components have configuration that modifies how they function. Below is a sample config opentelemetry protocol receiver config file:

```yaml
protocols:
  grpc:
    # The following entry demonstrates how to specify TLS credentials for the server.
    # Note: These files do not exist. If the receiver is started with this configuration, it will fail.
    tls:
      cert_file: test.crt
      key_file: test.key

    # The following demonstrates how to set maximum limits on stream, message size and connection idle time.
    # Note: The test yaml has demonstrated configuration on a grouped by their structure; however, all of the settings can
    # be mix and matched like adding the maximum connection idle setting in this example.
    max_recv_msg_size_mib: 32
    max_concurrent_streams: 16
    read_buffer_size: 1024
    write_buffer_size: 1024

    # The following entry configures all of the keep alive settings. These settings are used to configure the receiver.
    keepalive:
      server_parameters:
        max_connection_idle: 11s
        max_connection_age: 12s
        max_connection_age_grace: 13s
        time: 30s
        timeout: 5s
      enforcement_policy:
        min_time: 10s
        permit_without_stream: true
  http:
    # The following entry demonstrates how to specify TLS credentials for the server.
    # Note: These files do not exist. If the receiver is started with this configuration, it will fail.
    tls:
      cert_file: test.crt
      key_file: test.key

    # The following entry demonstrates how to configure the OTLP receiver to allow Cross-Origin Resource Sharing (CORS).
    # Both fully qualified domain names and the use of wildcards are supported.
    cors:
      allowed_origins:
        - https://*.test.com # Wildcard subdomain. Allows domains like https://www.test.com and https://foo.test.com but not https://wwwtest.com.
        - https://test.com # Fully qualified domain name. Allows https://test.com only.
      max_age: 7200

```
Link to source, [here](https://github.com/open-telemetry/opentelemetry-collector/blob/f64389d15f8b4dbddd807a16aabd84a57ce7826b/receiver/otlpreceiver/testdata/config.yaml)

Below is an initial config.yaml for the kubearmor receiver:

```yaml
# Proposwed ID for kubearmor receiver
kubearmorreceiver:
  #Specifies the kubearmor relay server endpoint, this would be optional by default it would be the value of the KUBEARMOR_SERVICE or in a k8 environment, the value
  # of the kubearmor relay service endpoint
  endpoint:https://127.0.0.1:32767 
  # By default all of kubearmor telemetry data would be forwarded but users can exclude any of them here. Accepted values are: logs, alert, visibility events
  exclude:
    -alerts
```
### Mechanism

The general mechanism is to create a client for the relay server in the code base, then create stanza operator input for the logs

### TODO

- Decide if this component should be part of the [opentelemetry collector contrib](https://github.com/open-telemetry/opentelemetry-collector-contrib). The collector contrib houses a number of open telemetry components from jaegar, AWS, datadog etc.
- Decide if to create a generic grpc stanza input operator for the collector contrib as part of the project.
