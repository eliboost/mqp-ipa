package ch.post.pf.mqpopsui.dto.jolokia;

import com.fasterxml.jackson.annotation.JsonAlias;
import com.fasterxml.jackson.annotation.JsonIgnoreProperties;
import lombok.Getter;
import lombok.Setter;
import lombok.ToString;

import java.util.List;
import java.util.Map;

@Getter
@Setter
@ToString
@JsonIgnoreProperties(ignoreUnknown = true)
public class JolokiaAMQMessage {
    private String address;
    private boolean durable;
    private Long expiration;
    private boolean largeMessage;
    private String messageID;
    private Long persistentSize;
    private Integer priority;
    private String protocol;
    private boolean redelivered;
    private String timestamp;
    private String type;
    private String text;
    private Integer deliveryCount;
    private boolean isPaged;
    @JsonAlias({"BodyPreview", "bodyPreview"})
    private List<Integer> bodyPreview;
    @JsonAlias({"StringProperties", "stringProperties"})
    private Map<String, String> stringProperties;
    @JsonAlias({"IntProperties", "intProperties"})
    private Map<String, Integer> intProperties;
    @JsonAlias({"LongProperties", "longProperties"})
    private Map<String, Long> longProperties;
    @JsonAlias({"ByteProperties", "byteProperties"})
    private Map<String, Byte> byteProperties;
    @JsonAlias({"BooleanProperties", "booleanProperties"})
    private Map<String, Boolean> booleanProperties;
    @JsonAlias({"DoubleProperties", "doubleProperties"})
    private Map<String, Double> doubleProperties;
    @JsonAlias({"FloatProperties", "floatProperties"})
    private Map<String, Float> floatProperties;
    @JsonAlias({"ShortProperties", "shortProperties"})
    private Map<String, Short> shortProperties;
}
