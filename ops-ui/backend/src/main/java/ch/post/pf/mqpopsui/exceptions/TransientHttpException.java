package ch.post.pf.mqpopsui.exceptions;


import java.io.IOException;

/**
 * Marks an HTTP failure that is worth retrying (e.g. a 5xx response from GitLab or
 * Backstage). Extends {@link IOException} so it is naturally treated as transient by the
 * retry predicate together with connect/read timeouts and connection resets.
 */
public class TransientHttpException extends IOException {
    public TransientHttpException(String message) {
        super(message);
    }
}
