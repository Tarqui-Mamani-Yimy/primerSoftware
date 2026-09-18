package com.aiuml.architect.project;
import jakarta.persistence.*; import org.springframework.data.jpa.repository.JpaRepository; import java.time.*; import java.util.*;
@Entity @Table(name="projects") class ProjectEntity { @Id UUID id; String name; String description; @Column(name="access_code") String accessCode; @Column(name="owner_id") UUID ownerId; @Column(name="created_at") Instant createdAt; protected ProjectEntity(){} }
@Entity @Table(name="project_memberships") @IdClass(ProjectMembershipId.class) class ProjectMembershipEntity { @Id @Column(name="project_id") UUID projectId; @Id @Column(name="user_id") UUID userId; String role; protected ProjectMembershipEntity(){} }
record ProjectMembershipId(UUID projectId,UUID userId) implements java.io.Serializable {}
interface ProjectRepository extends JpaRepository<ProjectEntity,UUID>{}
interface ProjectMembershipRepository extends JpaRepository<ProjectMembershipEntity,ProjectMembershipId>{ List<ProjectMembershipEntity> findByUserId(UUID userId); boolean existsByProjectIdAndUserId(UUID projectId,UUID userId); }
