package ch.post.pf.mqpopsui.dto.xml.weblogic;

import jakarta.xml.bind.annotation.*;
import lombok.Getter;
import lombok.Setter;

@Getter
@Setter
@XmlAccessorType(XmlAccessType.FIELD)
public class XmlProperty {

    @XmlAttribute
    private String name;

    @XmlElements({
            @XmlElement(name = "String", namespace = "http://www.bea.com/WLS/JMS/Message", type = String.class),
            @XmlElement(name = "Long", namespace = "http://www.bea.com/WLS/JMS/Message", type = Long.class),
            @XmlElement(name = "Int", namespace = "http://www.bea.com/WLS/JMS/Message", type = Integer.class)
    })
    private Object value;
}
