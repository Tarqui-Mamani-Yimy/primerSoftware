package com.aiuml.architect.auth;
import jakarta.persistence.*; import java.time.Instant; import java.util.*; import org.springframework.data.jpa.repository.JpaRepository;
@Entity @Table(name="users") class UserEntity { @Id UUID id; @Column(name="display_name") String displayName; String email; @Column(name="password_hash") String passwordHash; protected UserEntity(){} }
@Entity @Table(name="refresh_tokens") class RefreshTokenEntity { @Id UUID id; @Column(name="user_id") UUID userId; @Column(name="token_hash") String tokenHash; @Column(name="expires_at") Instant expiresAt; @Column(name="revoked_at") Instant revokedAt; @Column(name="created_at") Instant createdAt; protected RefreshTokenEntity(){} RefreshTokenEntity(UUID userId,String hash,Instant expiry){id=UUID.randomUUID();this.userId=userId;tokenHash=hash;expiresAt=expiry;createdAt=Instant.now();} }
interface UserRepository extends JpaRepository<UserEntity,UUID>{ Optional<UserEntity> findByEmailIgnoreCase(String email); }
interface RefreshTokenRepository extends JpaRepository<RefreshTokenEntity,UUID>{ Optional<RefreshTokenEntity> findByTokenHashAndRevokedAtIsNull(String hash); }
