package ch.post.pf.mqpopsui.dto.xml.weblogic;

import jakarta.xml.bind.annotation.XmlAccessType;
import jakarta.xml.bind.annotation.XmlAccessorType;
import jakarta.xml.bind.annotation.XmlElement;
import jakarta.xml.bind.annotation.XmlRootElement;
import lombok.Getter;
import lombok.Setter;

import java.util.List;

@Getter
@Setter
@XmlRootElement(name = "JMSMessageExport")
@XmlAccessorType(XmlAccessType.FIELD)
public class XmlMessageExport {

    @XmlElement(name = "WLJMSMessage", namespace = "http://www.bea.com/WLS/JMS/Message")
    private List<XmlWLJMSMessage> messages;

}
