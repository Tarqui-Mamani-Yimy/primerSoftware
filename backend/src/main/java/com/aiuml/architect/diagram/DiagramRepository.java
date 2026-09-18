package com.aiuml.architect.diagram;

import org.springframework.data.jpa.repository.JpaRepository;
import java.util.List;
import java.util.Optional;
import java.util.UUID;

public interface DiagramRepository extends JpaRepository<DiagramEntity, UUID> {
  List<DiagramEntity> findByProjectIdOrderByUpdatedAtDesc(UUID projectId);
  Optional<DiagramEntity> findByIdAndProjectId(UUID id, UUID projectId);
  long countByProjectId(UUID projectId);
}
