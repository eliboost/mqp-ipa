package ch.post.pf.mqpopsui.deserialize;

import com.fasterxml.jackson.core.JsonParser;
import com.fasterxml.jackson.databind.DeserializationContext;
import com.fasterxml.jackson.databind.JsonDeserializer;
import com.fasterxml.jackson.databind.JsonNode;

import java.io.IOException;
import java.util.ArrayList;
import java.util.List;

public class TextDeserializer extends JsonDeserializer<List<String>> {
    @Override
    public List<String> deserialize(JsonParser p, DeserializationContext ctxt) throws IOException {
        JsonNode node = p.getCodec().readTree(p);
        List<String> values = new ArrayList<>();

        if (node.isArray()) {
            for (JsonNode element : node) {
                values.add(element.asText());
            }
        } else if (node.isTextual() || node.isInt() || node.isBoolean()) {
            values.add(node.asText());
        }

        return values;
    }
}
