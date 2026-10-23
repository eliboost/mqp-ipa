package ch.post.pf.mqpopsui.dto.xml.weblogic;

import jakarta.xml.bind.annotation.XmlAccessType;
import jakarta.xml.bind.annotation.XmlAccessorType;
import jakarta.xml.bind.annotation.XmlElement;
import lombok.Getter;
import lombok.Setter;

@Getter
@Setter
@XmlAccessorType(XmlAccessType.FIELD)
public class XmlWLJMSMessage {

    @XmlElement(name = "Header", namespace = "http://www.bea.com/WLS/JMS/Message")
    private XmlHeader headers;

    @XmlElement(name = "Body", namespace = "http://www.bea.com/WLS/JMS/Message")
    private XmlBody body;

}
