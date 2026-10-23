package ch.post.pf.mqpopsui.dto.xml.artemis;

import jakarta.xml.bind.annotation.XmlAccessType;
import jakarta.xml.bind.annotation.XmlAccessorType;
import jakarta.xml.bind.annotation.XmlAttribute;
import lombok.Getter;
import lombok.Setter;

@Getter
@Setter
@XmlAccessorType(XmlAccessType.FIELD)
public class XmlProperty {

    @XmlAttribute
    private String name;

    @XmlAttribute
    private String value;

    @XmlAttribute
    private String type;
}
