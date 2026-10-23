package ch.post.pf.mqpopsui.dto.jolokia;

import ch.post.pf.mqpopsui.deserialize.AMQMessageDeserializer;
import com.fasterxml.jackson.annotation.JsonIgnoreProperties;
import com.fasterxml.jackson.databind.annotation.JsonDeserialize;
import lombok.Getter;
import lombok.Setter;
import lombok.ToString;

import java.util.List;

@JsonIgnoreProperties(ignoreUnknown = true)
@Getter
@Setter
@ToString
public class JolokiaAMQMessageResponse extends JolokiaBaseResponse {

    @JsonDeserialize(using = AMQMessageDeserializer.class)
    private List<JolokiaAMQMessage> value;
}
