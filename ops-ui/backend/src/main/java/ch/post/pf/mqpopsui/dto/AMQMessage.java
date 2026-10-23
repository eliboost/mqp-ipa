package ch.post.pf.mqpopsui.dto;

import com.fasterxml.jackson.annotation.JsonIgnoreProperties;
import com.fasterxml.jackson.annotation.JsonProperty;
import lombok.Getter;
import lombok.Setter;
import lombok.ToString;

import java.util.List;
import java.util.Map;

@Getter
@Setter
@ToString
@JsonIgnoreProperties(ignoreUnknown = true)
public class AMQMessage {

    private String text;
    private String messageID;
    private List<Integer> bodyPreview;
    private Map<String, AMQProperty> properties;
    private Map<String, String> headers;

    @JsonProperty("bodyPreview")
    public List<Integer> getBodyPreview() {
        return bodyPreview;
    }

    @JsonProperty("BodyPreview")
    public void setBodyPreview(List<Integer> bodyPreview) {
        this.bodyPreview = bodyPreview;
    }

}
