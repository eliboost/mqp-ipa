package ch.post.pf.mqpopsui.dto.xml.artemis;

import jakarta.xml.bind.annotation.*;
import lombok.Getter;
import lombok.Setter;

import java.util.List;

@Getter
@Setter
@XmlAccessorType(XmlAccessType.FIELD)
public class XmlMessage {

    @XmlAttribute
    private long id;

    @XmlAttribute
    private int priority;

    @XmlAttribute
    private long expiration;

    @XmlAttribute
    private long timestamp;

    @XmlAttribute
    private String type;

    @XmlAttribute(name = "user-id")
    private String userId;

    @XmlElementWrapper(name = "properties")
    @XmlElement(name = "property")
    private List<XmlProperty> properties;

    @XmlElementWrapper(name = "queues")
    @XmlElement(name = "queue")
    private List<XmlQueue> queues;

    @XmlElement
    private String body;
}
