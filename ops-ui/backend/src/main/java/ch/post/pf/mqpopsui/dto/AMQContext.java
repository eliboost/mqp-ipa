package ch.post.pf.mqpopsui.dto;

import com.fasterxml.jackson.annotation.JsonIgnore;
import lombok.Getter;
import lombok.Setter;

import java.util.List;

@Getter
@Setter
public class AMQContext {

    private String name;

    @JsonIgnore
    private List<AMQInstance> instances;
}