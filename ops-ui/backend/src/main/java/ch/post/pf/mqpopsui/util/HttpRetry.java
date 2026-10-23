package ch.post.pf.mqpopsui.util;

import ch.post.pf.mqpopsui.exceptions.TransientHttpException;
import lombok.extern.slf4j.Slf4j;

import java.io.IOException;
import java.io.InterruptedIOException;
import java.net.ConnectException;
import java.net.SocketException;
import java.net.SocketTimeoutException;
import java.net.UnknownHostException;
import java.util.concurrent.Callable;
import java.util.function.Predicate;

/**
 * Small, dependency-free retry helper for the raw {@link java.net.HttpURLConnection}
 * calls against GitLab and Backstage. Retries only on transient failures (connect/read
 * timeouts, connection resets, 5xx) so that a momentarily slow or flapping upstream does
 * not immediately fail an import or the scheduled fetch job. Permanent failures (4xx,
 * parsing errors) are not retried.
 */
@Slf4j
public final class HttpRetry {

    private HttpRetry() {
    }

    /**
     * Standard transient classifier: connect/read timeouts, connection resets and
     * {@link TransientHttpException} raised for 5xx. A plain {@link IOException} is used by
     * the callers to signal a permanent 4xx client error, so it is deliberately not treated
     * as transient; parsing errors are not transient either.
     */
    public static boolean isTransient(Exception e) {
        return e instanceof TransientHttpException
                || e instanceof SocketTimeoutException
                || e instanceof ConnectException
                || e instanceof UnknownHostException
                || e instanceof SocketException
                || e instanceof InterruptedIOException;
    }

    /**
     * Execute {@code action}, retrying when {@code retryable} accepts the thrown exception.
     *
     * @param maxAttempts    total number of attempts (>= 1); 1 means no retry
     * @param backoffMillis  delay between attempts, multiplied by the attempt number (linear backoff)
     * @param description    short label used for logging
     * @param retryable      decides whether a thrown exception is worth retrying
     * @param action         the call to execute
     */
    public static <T> T execute(int maxAttempts, long backoffMillis, String description,
                                Predicate<Exception> retryable, Callable<T> action) throws Exception {
        int attempts = Math.max(1, maxAttempts);
        Exception last = null;

        for (int attempt = 1; attempt <= attempts; attempt++) {
            try {
                return action.call();
            } catch (IOException e) {
                last = e;
                if (attempt >= attempts || !retryable.test(e)) {
                    throw e;
                }
                long sleep = backoffMillis * attempt;
                log.warn("Transient failure on '{}' (attempt {}/{}): {}. Retrying in {} ms.",
                        description, attempt, attempts, e.getMessage(), sleep);
                if (sleep > 0) {
                    try {
                        Thread.sleep(sleep);
                    } catch (InterruptedException ie) {
                        Thread.currentThread().interrupt();
                        throw e;
                    }
                }
            }
        }
        // Unreachable: the loop either returns or throws, but the compiler needs a terminal statement.
        throw last;
    }
}
