package ch.post.pf.mqpopsui.mapper;

import ch.post.pf.mqpopsui.dto.AMQMessage;
import ch.post.pf.mqpopsui.dto.AMQProperty;
import ch.post.pf.mqpopsui.dto.AMQPropertyType;
import ch.post.pf.mqpopsui.dto.jolokia.JolokiaAMQMessage;
import org.mapstruct.Mapper;
import org.mapstruct.Mapping;

import java.util.HashMap;
import java.util.List;
import java.util.Map;

@Mapper(componentModel = "spring")
public interface MessageMapper {

    @Mapping(target = "properties", expression = "java(mapProperties(message))")
    @Mapping(target = "headers", expression = "java(mapHeaders(message))")
    AMQMessage toAMQMessage(JolokiaAMQMessage message);

    List<AMQMessage> toAMQMessageList(List<JolokiaAMQMessage> messageList);

    default <T> void putAll(
            Map<String, AMQProperty> propertiesList,
            Map<String, T> source,
            AMQPropertyType type
    ) {
        if(source == null) return;

        source.forEach((key, value) -> {
            if(value != null) {
                propertiesList.put(key, new AMQProperty(String.valueOf(value), type));
            }
        });
    }

    default Map<String, String> mapHeaders(JolokiaAMQMessage message) {
        Map<String, String> headerList = new HashMap<>();

        headerList.put("address", message.getAddress());
        headerList.put("durable", String.valueOf(message.isDurable()));
        headerList.put("expiration", String.valueOf(message.getExpiration()));
        headerList.put("largeMessage", String.valueOf(message.isLargeMessage()));
        headerList.put("messageID", message.getMessageID());
        headerList.put("persistentSize", message.getPersistentSize() + " Bytes");
        headerList.put("priority", String.valueOf(message.getPriority()));
        headerList.put("protocol", message.getProtocol());
        headerList.put("redelivered", String.valueOf(message.isRedelivered()));
        headerList.put("timestamp", message.getTimestamp());
        headerList.put("type", message.getType());
        headerList.put("deliveryCount", String.valueOf(message.getDeliveryCount()));
        headerList.put("isPaged", String.valueOf(message.isPaged()));

        return headerList;
    }

    default Map<String, AMQProperty> mapProperties(JolokiaAMQMessage message) {
        Map<String, AMQProperty> propertiesList = new HashMap<>();

        putAll(propertiesList, message.getBooleanProperties(), AMQPropertyType.BOOLEAN);
        putAll(propertiesList, message.getByteProperties(), AMQPropertyType.BYTE);
        putAll(propertiesList, message.getShortProperties(), AMQPropertyType.SHORT);
        putAll(propertiesList, message.getIntProperties(), AMQPropertyType.INT);
        putAll(propertiesList, message.getLongProperties(), AMQPropertyType.LONG);
        putAll(propertiesList, message.getFloatProperties(), AMQPropertyType.FLOAT);
        putAll(propertiesList, message.getDoubleProperties(), AMQPropertyType.DOUBLE);
        putAll(propertiesList, message.getStringProperties(), AMQPropertyType.STRING);

        return propertiesList;
    }


}
