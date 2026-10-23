package ch.post.pf.mqpopsui.dto.jolokia;

import com.fasterxml.jackson.annotation.JsonIgnoreProperties;
import lombok.Getter;
import lombok.Setter;
import lombok.ToString;

@JsonIgnoreProperties(ignoreUnknown = true)
@Getter
@Setter
@ToString
public class JolokiaRequest {

    private String type;
    private String mbean;
    private String operation;
    private String[] arguments;
    private String component;
    private String address;
    private String subcomponent;
    private String queue;
    private String attribute;

}
