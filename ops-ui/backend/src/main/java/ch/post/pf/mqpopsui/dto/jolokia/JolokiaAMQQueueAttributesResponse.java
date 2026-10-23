package ch.post.pf.mqpopsui.dto.jolokia;

import ch.post.pf.mqpopsui.deserialize.AMQQueueAttributesDeserializer;
import ch.post.pf.mqpopsui.dto.AMQQueue;
import com.fasterxml.jackson.annotation.JsonIgnoreProperties;
import com.fasterxml.jackson.databind.annotation.JsonDeserialize;
import lombok.Getter;
import lombok.Setter;
import lombok.ToString;

@JsonIgnoreProperties(ignoreUnknown = true)
@Getter
@Setter
@ToString
public class JolokiaAMQQueueAttributesResponse extends JolokiaBaseResponse {

    @JsonDeserialize(using = AMQQueueAttributesDeserializer.class)
    private AMQQueue value;
}
