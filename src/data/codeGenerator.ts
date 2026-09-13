import { UMLClassNode, JpaStrategy, CodeFile } from '../types';

export function generateAllCodeFiles(classes: UMLClassNode[], strategy: JpaStrategy): CodeFile[] {
  const orderClass = classes.find(c => c.id === 'order') || classes[0];
  const customerClass = classes.find(c => c.id === 'customer') || classes[0];
  const paymentClass = classes.find(c => c.id === 'payment');

  const orderJava = generateOrderJava(orderClass, strategy);
  const sqlDdl = generatePostgresDdl(classes, strategy);
  const orderController = generateOrderControllerJava();
  const customerJava = generateCustomerJava(customerClass);
  const orderItemJava = generateOrderItemJava();
  const paymentJava = generatePaymentJava(paymentClass, strategy);
  const appYml = generateApplicationYml();
  const pomXml = generatePomXml();

  return [
    {
      id: 'order-java',
      path: 'src/main/java/com/architect/domain/model/Order.java',
      filename: 'Order.java',
      badge: '@Entity',
      badgeType: 'primary',
      language: 'java',
      content: orderJava
    },
    {
      id: 'sql-ddl',
      path: 'src/main/resources/db/migration/V1__init_schema.sql',
      filename: 'V1__init_schema.sql',
      badge: 'DDL',
      badgeType: 'tertiary',
      language: 'sql',
      content: sqlDdl
    },
    {
      id: 'order-controller-java',
      path: 'src/main/java/com/architect/domain/controller/OrderRestController.java',
      filename: 'OrderRestController.java',
      badge: 'REST',
      badgeType: 'secondary',
      language: 'java',
      content: orderController
    },
    {
      id: 'customer-java',
      path: 'src/main/java/com/architect/domain/model/Customer.java',
      filename: 'Customer.java',
      badge: 'Entity',
      badgeType: 'outline',
      language: 'java',
      content: customerJava
    },
    {
      id: 'order-item-java',
      path: 'src/main/java/com/architect/domain/model/OrderItem.java',
      filename: 'OrderItem.java',
      badge: 'Entity',
      badgeType: 'outline',
      language: 'java',
      content: orderItemJava
    },
    {
      id: 'payment-java',
      path: 'src/main/java/com/architect/domain/model/Payment.java',
      filename: 'Payment.java',
      badge: 'Entity',
      badgeType: 'outline',
      language: 'java',
      content: paymentJava
    },
    {
      id: 'application-yml',
      path: 'src/main/resources/application.yml',
      filename: 'application.yml',
      language: 'yaml',
      content: appYml
    },
    {
      id: 'pom-xml',
      path: 'pom.xml',
      filename: 'pom.xml',
      language: 'xml',
      content: pomXml
    }
  ];
}

function generateOrderJava(order: UMLClassNode, _strategy: JpaStrategy): string {
  const customAttrs = order.attributes
    .filter(a => !['id', 'status', 'totalAmount', 'createdAt'].includes(a.name))
    .map(a => `    private ${a.type} ${a.name};`)
    .join('\n');

  return `package com.architect.domain.model;

import jakarta.persistence.*;
import jakarta.validation.constraints.*;
import lombok.*;
import java.math.BigDecimal;
import java.time.Instant;
import java.util.*;

@Entity
@Table(name = "${order.tableBinding || 't_orders'}", indexes = {
    @Index(name = "idx_order_customer_id", columnList = "customer_id"),
    @Index(name = "idx_order_created_at", columnList = "created_at DESC")
})
@Getter
@Setter
@NoArgsConstructor
@AllArgsConstructor
@Builder
public class Order {

    @Id
    @GeneratedValue(strategy = GenerationType.UUID)
    private UUID id;

    @NotNull
    @ManyToOne(fetch = FetchType.LAZY, optional = false)
    @JoinColumn(name = "customer_id", nullable = false, foreignKey = @ForeignKey(name = "fk_orders_customer"))
    private Customer customer;

    @Builder.Default
    @OneToMany(mappedBy = "order", cascade = CascadeType.ALL, orphanRemoval = true)
    private List<OrderItem> items = new ArrayList<>();

    @Enumerated(EnumType.STRING)
    @Column(name = "order_status", nullable = false, length = 32)
    private OrderStatus status;

    @NotNull
    @PositiveOrZero
    @Column(name = "total_amount", precision = 12, scale = 2, nullable = false)
    private BigDecimal totalAmount;

    @Column(name = "created_at", nullable = false, updatable = false)
    private Instant createdAt = Instant.now();
${customAttrs ? '\n' + customAttrs : ''}
}`;
}

function generatePostgresDdl(classes: UMLClassNode[], strategy: JpaStrategy): string {
  const inheritanceComment = strategy === 'JOINED' 
    ? '-- JPA Inheritance: JOINED (separate tables with foreign key to parent)'
    : strategy === 'SINGLE'
    ? '-- JPA Inheritance: SINGLE_TABLE (single table with discriminator column)'
    : '-- JPA Inheritance: TABLE_PER_CLASS (concrete table per subclass with all inherited columns)';

  return `-- PostgreSQL 16 DDL Migration Generated from UML Architecture
-- Target: Spring Boot 3.2.3 • Flyway v10
${inheritanceComment}

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE t_customers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) NOT NULL UNIQUE,
    first_name VARCHAR(100) NOT NULL,
    last_name VARCHAR(100) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE t_orders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    customer_id UUID NOT NULL REFERENCES t_customers(id) ON DELETE RESTRICT,
    order_status VARCHAR(32) NOT NULL DEFAULT 'PENDING',
    total_amount NUMERIC(12, 2) NOT NULL CHECK (total_amount >= 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE t_order_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id UUID NOT NULL REFERENCES t_orders(id) ON DELETE CASCADE,
    sku VARCHAR(64) NOT NULL,
    quantity INT NOT NULL CHECK (quantity > 0),
    unit_price NUMERIC(10, 2) NOT NULL CHECK (unit_price >= 0)
);

-- Payment Hierarchy (${strategy})
${strategy === 'SINGLE' ? `
CREATE TABLE t_payments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id UUID REFERENCES t_orders(id) ON DELETE RESTRICT,
    dtype VARCHAR(31) NOT NULL,
    amount NUMERIC(12, 2) NOT NULL CHECK (amount >= 0),
    transaction_id VARCHAR(128) NOT NULL,
    card_number_mask VARCHAR(20),
    auth_code VARCHAR(64),
    stripe_payment_intent_id VARCHAR(128),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
` : `
CREATE TABLE t_payments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id UUID REFERENCES t_orders(id) ON DELETE RESTRICT,
    amount NUMERIC(12, 2) NOT NULL CHECK (amount >= 0),
    transaction_id VARCHAR(128) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE t_payments_cc (
    payment_id UUID PRIMARY KEY REFERENCES t_payments(id) ON DELETE CASCADE,
    card_number_mask VARCHAR(20) NOT NULL,
    auth_code VARCHAR(64) NOT NULL
);

CREATE TABLE t_payments_stripe (
    payment_id UUID PRIMARY KEY REFERENCES t_payments(id) ON DELETE CASCADE,
    stripe_payment_intent_id VARCHAR(128) NOT NULL
);
`}

CREATE INDEX idx_orders_customer_id ON t_orders(customer_id);
CREATE INDEX idx_orders_created_at ON t_orders(created_at DESC);
CREATE INDEX idx_order_items_order_id ON t_order_items(order_id);
`;
}

function generateOrderControllerJava(): string {
  return `package com.architect.domain.controller;

import com.architect.domain.dto.*;
import com.architect.domain.service.OrderService;
import jakarta.validation.Valid;
import lombok.RequiredArgsConstructor;
import org.springframework.http.*;
import org.springframework.web.bind.annotation.*;
import java.util.UUID;

@RestController
@RequestMapping("/api/v1/orders")
@RequiredArgsConstructor
public class OrderRestController {

    private final OrderService orderService;

    @PostMapping
    public ResponseEntity<OrderResponseDTO> createOrder(@Valid @RequestBody OrderRequestDTO request) {
        return ResponseEntity.status(HttpStatus.CREATED).body(orderService.create(request));
    }

    @GetMapping("/{id}")
    public ResponseEntity<OrderResponseDTO> getOrder(@PathVariable UUID id) {
        return ResponseEntity.ok(orderService.findById(id));
    }
}`;
}

function generateCustomerJava(cust: UMLClassNode): string {
  return `package com.architect.domain.model;

import jakarta.persistence.*;
import jakarta.validation.constraints.*;
import lombok.*;
import java.util.UUID;

@Entity
@Table(name = "${cust.tableBinding || 't_customers'}")
@Getter
@Setter
@NoArgsConstructor
@AllArgsConstructor
@Builder
public class Customer {

    @Id
    @GeneratedValue(strategy = GenerationType.UUID)
    private UUID id;

    @Email
    @NotNull
    private String email;

    @NotNull
    private String firstName;

    @NotNull
    private String lastName;
}`;
}

function generateOrderItemJava(): string {
  return `package com.architect.domain.model;

import jakarta.persistence.*;
import jakarta.validation.constraints.*;
import lombok.*;
import java.math.BigDecimal;
import java.util.UUID;

@Entity
@Table(name = "t_order_items")
@Getter
@Setter
@NoArgsConstructor
@AllArgsConstructor
@Builder
public class OrderItem {

    @Id
    @GeneratedValue(strategy = GenerationType.UUID)
    private UUID id;

    @ManyToOne(fetch = FetchType.LAZY)
    @JoinColumn(name = "order_id", nullable = false)
    private Order order;

    @NotNull
    private String sku;

    @Positive
    private Integer quantity;

    @PositiveOrZero
    private BigDecimal unitPrice;

    public BigDecimal getSubtotal() {
        return unitPrice.multiply(BigDecimal.valueOf(quantity));
    }
}`;
}

function generatePaymentJava(_payment?: UMLClassNode, strategy: JpaStrategy = 'JOINED'): string {
  return `package com.architect.domain.model;

import jakarta.persistence.*;
import jakarta.validation.constraints.*;
import lombok.*;
import java.math.BigDecimal;
import java.util.UUID;

@Entity
@Table(name = "t_payments")
@Inheritance(strategy = InheritanceType.${strategy})
@Getter
@Setter
@NoArgsConstructor
@AllArgsConstructor
public abstract class Payment {

    @Id
    @GeneratedValue(strategy = GenerationType.UUID)
    protected UUID id;

    @NotNull
    protected String transactionId;

    @NotNull
    @PositiveOrZero
    protected BigDecimal amount;

    public abstract PaymentStatus execute();
}`;
}

function generateApplicationYml(): string {
  return `spring:
  application:
    name: spring-boot-ecommerce-core
  datasource:
    url: jdbc:postgresql://localhost:5432/ecommerce_db
    username: \${DB_USERNAME:postgres}
    password: \${DB_PASSWORD:postgres}
    driver-class-name: org.postgresql.Driver
  jpa:
    hibernate:
      ddl-auto: validate
    properties:
      hibernate:
        format_sql: true
        dialect: org.hibernate.dialect.PostgreSQLDialect
  flyway:
    enabled: true
    locations: classpath:db/migration

server:
  port: 8080
`;
}

function generatePomXml(): string {
  return `<?xml version="1.0" encoding="UTF-8"?>
<project xmlns="http://maven.apache.org/POM/4.0.0" 
         xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
         xsi:schemaLocation="http://maven.apache.org/POM/4.0.0 https://maven.apache.org/xsd/maven-4.0.0.xsd">
    <modelVersion>4.0.0</modelVersion>
    <parent>
        <groupId>org.springframework.boot</groupId>
        <artifactId>spring-boot-starter-parent</artifactId>
        <version>3.2.3</version>
        <relativePath/>
    </parent>
    <groupId>com.architect</groupId>
    <artifactId>spring-boot-ecommerce-core</artifactId>
    <version>0.0.1-SNAPSHOT</version>
    <name>spring-boot-ecommerce-core</name>
    <description>Generated domain architecture with AI UML</description>
    <properties>
        <java.version>21</java.version>
    </properties>
    <dependencies>
        <dependency>
            <groupId>org.springframework.boot</groupId>
            <artifactId>spring-boot-starter-web</artifactId>
        </dependency>
        <dependency>
            <groupId>org.springframework.boot</groupId>
            <artifactId>spring-boot-starter-data-jpa</artifactId>
        </dependency>
        <dependency>
            <groupId>org.springframework.boot</groupId>
            <artifactId>spring-boot-starter-validation</artifactId>
        </dependency>
        <dependency>
            <groupId>org.postgresql</groupId>
            <artifactId>postgresql</artifactId>
            <scope>runtime</scope>
        </dependency>
        <dependency>
            <groupId>org.flywaydb</groupId>
            <artifactId>flyway-core</artifactId>
        </dependency>
        <dependency>
            <groupId>org.projectlombok</groupId>
            <artifactId>lombok</artifactId>
            <optional>true</optional>
        </dependency>
    </dependencies>
    <build>
        <plugins>
            <plugin>
                <groupId>org.springframework.boot</groupId>
                <artifactId>spring-boot-maven-plugin</artifactId>
            </plugin>
        </plugins>
    </build>
</project>`;
}
