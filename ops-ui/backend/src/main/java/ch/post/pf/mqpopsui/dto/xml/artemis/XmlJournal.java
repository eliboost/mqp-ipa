package ch.post.pf.mqpopsui.dto.xml.artemis;

import jakarta.xml.bind.annotation.*;
import lombok.Getter;
import lombok.Setter;

import java.util.List;

@Getter
@Setter
@XmlRootElement(name = "activemq-journal")
@XmlAccessorType(XmlAccessType.FIELD)
public class XmlJournal {

    @XmlElementWrapper(name = "messages")
    @XmlElement(name = "message")
    private List<XmlMessage> messages;
}
