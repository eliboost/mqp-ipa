package ch.postfinance.activemq.monitor;

import io.micrometer.core.instrument.Counter;
import io.micrometer.core.instrument.MeterRegistry;
import io.micrometer.core.instrument.Timer;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.stereotype.Component;

@Component
public class Metrics {
  public final Counter totalMessagesSentCounter;
  public final Counter totalMessagesConsumedCounter;
  public final Counter totalMessagesConsumedErrorsCounter;
  public final Counter totalMessagesDelayedCounter;

  public final Timer histogramDuration;

  @Autowired
  public Metrics(MeterRegistry meterRegistry) {
    this.totalMessagesSentCounter = Counter.builder("monitor_messages_sent_total")
        .description("Number of total sent messages")
        .register(meterRegistry);
    this.totalMessagesConsumedCounter = Counter.builder("monitor_messages_consumed_total")
        .description("Number of total consumed messages")
        .register(meterRegistry);
    this.totalMessagesConsumedErrorsCounter = Counter.builder("monitor_messages_consumed_errors_total")
        .description("Number of total errors consumed messages")
        .register(meterRegistry);
    this.totalMessagesDelayedCounter = Counter.builder("monitor_messages_delayed_total")
        .description("Number of total delayed messages")
        .register(meterRegistry);
    this.histogramDuration = Timer.builder("monitor_e2e_duration")
        .publishPercentiles(0.5, 0.95, 0.99)
        .description("Number of total consumed messages")
        .register(meterRegistry);

    // init rare metric series
    this.totalMessagesConsumedCounter.increment(0);
    this.totalMessagesConsumedErrorsCounter.increment(0);
  }
}
