package ch.post.pf.mqpopsui.dto.jolokia;

import ch.post.pf.mqpopsui.deserialize.TextDeserializer;
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
public class JolokiaStringResponse extends JolokiaBaseResponse {

    @JsonDeserialize(using = TextDeserializer.class)
    private List<String> value;
}
