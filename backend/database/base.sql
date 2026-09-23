CREATE TABLE usuarios (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    nombre          VARCHAR(150) NOT NULL,
    email           VARCHAR(150) NOT NULL UNIQUE,
    password_hash   VARCHAR(255) NOT NULL,
    activo          BOOLEAN NOT NULL DEFAULT true,
    fecha_creacion  TIMESTAMP NOT NULL DEFAULT now()
);

CREATE TABLE refresh_tokens (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    usuario_id          UUID NOT NULL REFERENCES usuarios(id) ON DELETE CASCADE,
    token               VARCHAR(500) NOT NULL UNIQUE,
    revocado            BOOLEAN NOT NULL DEFAULT false,
    fecha_expiracion    TIMESTAMP NOT NULL,
    fecha_creacion      TIMESTAMP NOT NULL DEFAULT now()
);

CREATE INDEX idx_refresh_tokens_usuario ON refresh_tokens(usuario_id);

-- ============================================================
-- ROLES Y PERMISOS (RBAC a nivel de plataforma)
-- ============================================================
CREATE TABLE roles (
    id              SERIAL PRIMARY KEY,
    nombre          VARCHAR(50) NOT NULL UNIQUE,   -- ej: 'ADMIN', 'USUARIO'
    descripcion     VARCHAR(255)
);

CREATE TABLE permisos (
    id              SERIAL PRIMARY KEY,
    nombre          VARCHAR(100) NOT NULL UNIQUE,  -- ej: 'PROYECTO_CREAR', 'PROYECTO_ELIMINAR'
    descripcion     VARCHAR(255)
);

CREATE TABLE rol_permisos (
    rol_id          INT NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    permiso_id      INT NOT NULL REFERENCES permisos(id) ON DELETE CASCADE,
    PRIMARY KEY (rol_id, permiso_id)
);

CREATE TABLE usuario_roles (
    usuario_id      UUID NOT NULL REFERENCES usuarios(id) ON DELETE CASCADE,
    rol_id          INT NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    PRIMARY KEY (usuario_id, rol_id)
);

-- ============================================================
-- PROYECTOS (equivalente a una "clase" de Google Classroom)
-- ============================================================
CREATE TABLE proyectos (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    nombre          VARCHAR(150) NOT NULL,
    descripcion     TEXT,
    codigo_acceso   VARCHAR(8) NOT NULL UNIQUE,
    id_creador      UUID NOT NULL REFERENCES usuarios(id) ON DELETE RESTRICT,
    fecha_creacion  TIMESTAMP NOT NULL DEFAULT now()
);

CREATE INDEX idx_proyectos_codigo_acceso ON proyectos(codigo_acceso);
CREATE INDEX idx_proyectos_creador ON proyectos(id_creador);

-- Rol dentro de un proyecto específico (distinto del rol global de la plataforma)
CREATE TYPE rol_proyecto AS ENUM ('PROPIETARIO', 'COLABORADOR');

CREATE TABLE proyecto_usuarios (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    proyecto_id     UUID NOT NULL REFERENCES proyectos(id) ON DELETE CASCADE,
    usuario_id      UUID NOT NULL REFERENCES usuarios(id) ON DELETE CASCADE,
    rol             rol_proyecto NOT NULL DEFAULT 'COLABORADOR',
    fecha_union     TIMESTAMP NOT NULL DEFAULT now(),

    CONSTRAINT uq_proyecto_usuario UNIQUE (proyecto_id, usuario_id)
);

CREATE INDEX idx_proyecto_usuarios_proyecto ON proyecto_usuarios(proyecto_id);
CREATE INDEX idx_proyecto_usuarios_usuario ON proyecto_usuarios(usuario_id);

-- ============================================================
-- DIAGRAMAS
-- ============================================================
CREATE TABLE diagramas (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    proyecto_id     UUID NOT NULL REFERENCES proyectos(id) ON DELETE CASCADE,
    nombre          VARCHAR(150) NOT NULL,
    fecha_creacion  TIMESTAMP NOT NULL DEFAULT now()
);

CREATE INDEX idx_diagramas_proyecto ON diagramas(proyecto_id);

-- ============================================================
-- DIAGRAMA_VERSIONES (historial tipo "snapshot")
-- ============================================================
CREATE TABLE diagrama_versiones (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    diagrama_id     UUID NOT NULL REFERENCES diagramas(id) ON DELETE CASCADE,
    snapshot_json   JSONB NOT NULL,
    autor_id        UUID NOT NULL REFERENCES usuarios(id) ON DELETE RESTRICT,
    mensaje         VARCHAR(255),
    fecha           TIMESTAMP NOT NULL DEFAULT now()
);

CREATE INDEX idx_diagrama_versiones_diagrama ON diagrama_versiones(diagrama_id);
CREATE INDEX idx_diagrama_versiones_snapshot_gin ON diagrama_versiones USING GIN (snapshot_json);

-- ============================================================
-- SESIONES DE TRABAJO (edición colaborativa en vivo sobre un diagrama)
-- ============================================================
CREATE TABLE sesiones_trabajo (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    diagrama_id     UUID NOT NULL REFERENCES diagramas(id) ON DELETE CASCADE,
    creado_por      UUID NOT NULL REFERENCES usuarios(id) ON DELETE RESTRICT,
    fecha_inicio    TIMESTAMP NOT NULL DEFAULT now(),
    fecha_fin       TIMESTAMP
);

CREATE INDEX idx_sesiones_trabajo_diagrama ON sesiones_trabajo(diagrama_id);

CREATE TABLE sesion_participantes (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    sesion_id       UUID NOT NULL REFERENCES sesiones_trabajo(id) ON DELETE CASCADE,
    usuario_id      UUID NOT NULL REFERENCES usuarios(id) ON DELETE CASCADE,
    fecha_union     TIMESTAMP NOT NULL DEFAULT now(),
    fecha_salida    TIMESTAMP,

    CONSTRAINT uq_sesion_usuario UNIQUE (sesion_id, usuario_id)
);

CREATE INDEX idx_sesion_participantes_sesion ON sesion_participantes(sesion_id);
CREATE INDEX idx_sesion_participantes_usuario ON sesion_participantes(usuario_id);

-- ============================================================
-- Datos base de roles y permisos (opcional, útil como seed mínimo)
-- ============================================================
INSERT INTO roles (nombre, descripcion) VALUES
('ADMIN', 'Administrador de la plataforma'),
('USUARIO', 'Usuario estándar');

INSERT INTO permisos (nombre, descripcion) VALUES
('PROYECTO_CREAR', 'Puede crear proyectos'),
('PROYECTO_ELIMINAR', 'Puede eliminar cualquier proyecto'),
('USUARIO_GESTIONAR', 'Puede administrar cuentas de usuario');

-- ADMIN tiene todos los permisos; USUARIO solo puede crear proyectos
INSERT INTO rol_permisos (rol_id, permiso_id)
SELECT r.id, p.id FROM roles r, permisos p WHERE r.nombre = 'ADMIN';

INSERT INTO rol_permisos (rol_id, permiso_id)
SELECT r.id, p.id FROM roles r, permisos p
WHERE r.nombre = 'USUARIO' AND p.nombre = 'PROYECTO_CREAR';