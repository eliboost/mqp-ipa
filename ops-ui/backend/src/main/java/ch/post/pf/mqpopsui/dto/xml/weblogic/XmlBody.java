package ch.post.pf.mqpopsui.dto.xml.weblogic;

import jakarta.xml.bind.annotation.XmlAccessType;
import jakarta.xml.bind.annotation.XmlAccessorType;
import jakarta.xml.bind.annotation.XmlElement;
import jakarta.xml.bind.annotation.XmlElements;
import lombok.Getter;
import lombok.Setter;

@Getter
@Setter
@XmlAccessorType(XmlAccessType.FIELD)
public class XmlBody {

    @XmlElements(
            @XmlElement(name = "Text", namespace = "http://www.bea.com/WLS/JMS/Message", type = String.class)
    )
    private Object textValue;

    @XmlElements(
            @XmlElement(name = "Bytes", namespace = "http://www.bea.com/WLS/JMS/Message", type = String.class)
    )
    private Object byteValue;


}
