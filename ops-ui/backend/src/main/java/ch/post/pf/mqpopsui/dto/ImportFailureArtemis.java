package ch.post.pf.mqpopsui.dto;

import ch.post.pf.mqpopsui.dto.xml.artemis.XmlMessage;

public record ImportFailureArtemis(int index, XmlMessage message, String error) {
}