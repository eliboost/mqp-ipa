package ch.post.pf.mqpopsui.dto.jolokia;

import com.fasterxml.jackson.annotation.JsonIgnoreProperties;
import lombok.Getter;
import lombok.Setter;

@JsonIgnoreProperties(ignoreUnknown = true)
@Getter
@Setter
public class JolokiaBaseResponse {

    private int status;
    private String error;

    public boolean isSuccess() {
        return status == 200;
    }
}
