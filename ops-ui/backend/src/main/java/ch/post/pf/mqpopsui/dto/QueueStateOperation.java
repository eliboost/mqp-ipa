package ch.post.pf.mqpopsui.dto;

import com.fasterxml.jackson.annotation.JsonCreator;

import java.util.Arrays;

public enum QueueStateOperation {
    PAUSE("pause"),
    RESUME("resume"),
    ENABLE("enable"),
    DISABLE("disable");

    private final String jolokiaOperation;

    QueueStateOperation(String jolokiaOperation) {
        this.jolokiaOperation = jolokiaOperation;
    }

    /** Signatur-lose Jolokia-EXEC-Operation, z.B. "pause()". */
    public String toJolokiaOperation() {
        return jolokiaOperation + "()";
    }

    @JsonCreator
    public static QueueStateOperation fromValue(String value) {
        return Arrays.stream(values())
                .filter(op -> op.jolokiaOperation.equalsIgnoreCase(value == null ? null : value.trim()))
                .findFirst()
                .orElseThrow(() -> new IllegalArgumentException("Unsupported queue state operation: " + value));
    }
}