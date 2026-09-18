package com.aiuml.architect.diagram;

import org.springframework.data.jpa.repository.JpaRepository;
import java.util.List;
import java.util.Optional;
import java.util.UUID;

interface DiagramVersionRepository extends JpaRepository<DiagramVersionEntity, UUID> {
  List<DiagramVersionEntity> findByDiagramIdOrderByVersionNumberDesc(UUID id);
  Optional<DiagramVersionEntity> findByDiagramIdAndVersionNumber(UUID id, int n);
  Optional<DiagramVersionEntity> findFirstByDiagramIdOrderByVersionNumberDesc(UUID id);
}
