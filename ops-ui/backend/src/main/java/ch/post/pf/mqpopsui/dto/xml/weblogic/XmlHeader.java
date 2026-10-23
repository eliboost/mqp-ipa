package ch.post.pf.mqpopsui.dto.xml.weblogic;

import jakarta.xml.bind.annotation.XmlAccessType;
import jakarta.xml.bind.annotation.XmlAccessorType;
import jakarta.xml.bind.annotation.XmlElement;
import jakarta.xml.bind.annotation.XmlElementWrapper;
import lombok.Getter;
import lombok.Setter;

import java.util.List;

@Getter
@Setter
@XmlAccessorType(XmlAccessType.FIELD)
public class XmlHeader {

    @XmlElement(name = "JMSMessageID", namespace = "http://www.bea.com/WLS/JMS/Message")
    private String messageId;

    @XmlElement(name = "JMSDeliveryMode", namespace = "http://www.bea.com/WLS/JMS/Message")
    private String deliveryMode;

    @XmlElement(name = "JMSDestination", namespace = "http://www.bea.com/WLS/JMS/Message")
    private XmlJMSDestination jmsDestination;

    @XmlElement(name = "JMSExpiration", namespace = "http://www.bea.com/WLS/JMS/Message")
    private Long expiration;

    @XmlElement(name = "JMSPriority", namespace = "http://www.bea.com/WLS/JMS/Message")
    private Integer priority;

    @XmlElement(name = "JMSRedelivered", namespace = "http://www.bea.com/WLS/JMS/Message")
    private boolean redelivered;

    @XmlElement(name = "JMSTimestamp", namespace = "http://www.bea.com/WLS/JMS/Message")
    private Long timestamp;

    @XmlElementWrapper(name = "Properties", namespace = "http://www.bea.com/WLS/JMS/Message")
    @XmlElement(name = "property", namespace = "http://www.bea.com/WLS/JMS/Message")
    private List<XmlProperty> properties;


}
