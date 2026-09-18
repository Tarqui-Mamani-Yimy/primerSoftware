package com.aiuml.architect.diagram;
import org.junit.jupiter.api.Test; import java.util.*; import static org.assertj.core.api.Assertions.assertThat;
class DiagramServiceTest { @Test void diagramDocumentKeepsCanonicalVersionedShape(){var d=new DiagramDocument(1,UUID.randomUUID(),"Orders",List.of(),List.of());assertThat(d.schemaVersion()).isEqualTo(1);assertThat(d.classes()).isEmpty();assertThat(d.relationships()).isEmpty();} }
