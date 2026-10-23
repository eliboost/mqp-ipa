package ch.post.pf.mqpopsui.jobs;

import ch.post.pf.mqpopsui.services.InstanceFetcherService;
import lombok.AllArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.scheduling.annotation.EnableScheduling;
import org.springframework.scheduling.annotation.Scheduled;
import org.springframework.stereotype.Component;

@Component
@Slf4j
@AllArgsConstructor
@EnableScheduling
public class InstanceFetcher {

    private static final int FIXED_DELAY = 1000 * 60 * 60;
    private static final int INITIAL_DELAY = 0;
    private InstanceFetcherService instanceFetcherService;

    @Scheduled(fixedDelay = FIXED_DELAY, initialDelay = INITIAL_DELAY)
    public void execute() {
        log.info("InstanceFetcher Job Started");

        try {
            instanceFetcherService.fetchInstances();
        } catch (Exception e) {
            log.error("InstanceFetcher Job failed", e);
        }
    }
}
