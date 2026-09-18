package com.aiuml.architect.diagram;

import com.fasterxml.jackson.databind.ObjectMapper;
import org.junit.jupiter.api.Test;
import java.util.List;
import static org.assertj.core.api.Assertions.assertThat;

class DiagramDocumentJsonTest {
  private final ObjectMapper mapper = new ObjectMapper();
  @Test void preservesFrontendShapeIncludingPackageAndPositions() throws Exception {
    var document = new DiagramDocument(1, null, "Orders", List.of(new DiagramDocument.UmlClass("order", "Order", "«Entity»", "com.example", "orders", 10, 20, 240, List.of(), List.of())), List.of());
    String json = mapper.writeValueAsString(document);
    assertThat(json).contains("\"package\":\"com.example\"").contains("\"x\":10").contains("\"y\":20");
    assertThat(mapper.readValue(json, DiagramDocument.class).classes().getFirst().packageName()).isEqualTo("com.example");
  }
}
