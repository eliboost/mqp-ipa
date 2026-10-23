package ch.post.pf.mqpopsui.dto;

import lombok.AllArgsConstructor;
import lombok.Getter;
import lombok.Setter;

@AllArgsConstructor
@Getter
@Setter
public class AMQProperty {
    String value;
    AMQPropertyType type;
}
