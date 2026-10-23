package ch.post.pf.mqpopsui.dto;

import ch.post.pf.mqpopsui.dto.xml.weblogic.XmlWLJMSMessage;

public record ImportFailureWeblogic(int index, XmlWLJMSMessage message, String error) {
}
