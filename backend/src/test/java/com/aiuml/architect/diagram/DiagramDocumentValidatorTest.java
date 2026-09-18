package com.aiuml.architect.diagram;

import static org.assertj.core.api.Assertions.assertThat;

import java.util.List;
import org.junit.jupiter.api.Test;

class DiagramDocumentValidatorTest {
  private final DiagramDocumentValidator validator = new DiagramDocumentValidator();

  @Test
  void acceptsAValidRelationshipDocument() {
    assertThat(validator.validate(document(relationship("order", "customer", "association", "1", "0..*")))).isEmpty();
  }

  @Test
  void rejectsMissingEndpointsUnsupportedTypesAndDuplicates() {
    assertThat(validator.validate(document(relationship("order", "missing", "association", null, null))))
        .anyMatch(error -> error.contains("missing class"));
    assertThat(validator.validate(document(relationship("order", "customer", "invalid", null, null))))
        .anyMatch(error -> error.contains("unsupported type"));
    var first = relationship("order", "customer", "association", null, null);
    var duplicate = new DiagramDocument.Relationship("rel-2", "order", "customer", "association", null, null, null);
    assertThat(validator.validate(document(first, duplicate))).anyMatch(error -> error.contains("duplicate"));
  }

  @Test
  void rejectsInvalidMultiplicityAndInvalidRealization() {
    assertThat(validator.validate(document(relationship("order", "customer", "association", "one", "0..*"))))
        .anyMatch(error -> error.contains("invalid source multiplicity"));
    assertThat(validator.validate(document(relationship("customer", "order", "realization", null, null))))
        .anyMatch(error -> error.contains("Realization"));
  }

  private DiagramDocument document(DiagramDocument.Relationship... relationships) {
    return new DiagramDocument(1, null, "Orders", List.of(
        new DiagramDocument.UmlClass("order", "Order", "«Entity»", null, null, 0, 0, null, List.of(), List.of()),
        new DiagramDocument.UmlClass("customer", "Customer", "«Interface»", null, null, 0, 0, null, List.of(), List.of())
    ), List.of(relationships));
  }

  private DiagramDocument.Relationship relationship(String source, String target, String type, String sourceMultiplicity, String targetMultiplicity) {
    return new DiagramDocument.Relationship("rel-1", source, target, type, sourceMultiplicity, targetMultiplicity, null);
  }
}
