package ch.post.pf.mqpopsui.dto.xml.weblogic;

import jakarta.xml.bind.annotation.XmlAccessType;
import jakarta.xml.bind.annotation.XmlAccessorType;
import jakarta.xml.bind.annotation.XmlAttribute;
import lombok.Getter;
import lombok.Setter;

@Getter
@Setter
@XmlAccessorType(XmlAccessType.FIELD)
public class XmlDestination {

    @XmlAttribute
    private String name;

    @XmlAttribute
    private String serverName;

    @XmlAttribute
    private String applicationName;

    @XmlAttribute
    private String moduleName;

}
