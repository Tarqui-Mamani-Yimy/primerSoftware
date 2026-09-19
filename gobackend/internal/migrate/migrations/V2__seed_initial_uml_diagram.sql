-- Populate the diagram seeded by V1 only when it still has V1's untouched empty model.
-- A full document comparison prevents replacing diagrams that users have already edited.
WITH updated_diagram AS (
  UPDATE diagrams
  SET document = $json$
{
  "schemaVersion": 1,
  "id": "cccccccc-cccc-cccc-cccc-cccccccccccc",
  "name": "Diagrama principal",
  "classes": [
    {
      "id": "customer",
      "name": "Customer",
      "stereotype": "«Entity»",
      "package": "com.nexus.orders",
      "tableBinding": "t_customers",
      "badge": "1..*",
      "x": 40,
      "y": 100,
      "width": 200,
      "attributes": [
        { "id": "c_id", "name": "id", "type": "UUID", "visibility": "+", "isPk": true, "annotations": ["@Id", "@GeneratedValue(UUID)"] },
        { "id": "c_email", "name": "email", "type": "String", "visibility": "+", "annotations": ["@Email", "@NotNull"] },
        { "id": "c_pwd", "name": "passwordHash", "type": "String", "visibility": "-", "annotations": ["@NotNull"] }
      ],
      "methods": [{ "id": "c_m1", "name": "verifyKyc()", "returnType": "Boolean", "visibility": "+" }]
    },
    {
      "id": "order",
      "name": "Order",
      "stereotype": "«Entity, AggregateRoot»",
      "package": "com.nexus.orders",
      "tableBinding": "t_orders",
      "badge": "t_orders",
      "x": 360,
      "y": 64,
      "width": 290,
      "attributes": [
        { "id": "o_id", "name": "id", "type": "UUID", "visibility": "+", "isPk": true, "annotations": ["@Id", "@GeneratedValue"] },
        { "id": "o_status", "name": "status", "type": "OrderStatus", "visibility": "+", "annotations": ["@Enumerated(STRING)"] },
        { "id": "o_total", "name": "totalAmount", "type": "BigDecimal", "visibility": "+", "annotations": ["precision=12, scale=2", "@NotNull", "@PositiveOrZero"] },
        { "id": "o_created", "name": "createdAt", "type": "Instant", "visibility": "#", "annotations": ["@Column(updatable = false)"] }
      ],
      "methods": [
        { "id": "o_m1", "name": "calculateTax(rate)", "returnType": "BigDecimal", "visibility": "+" },
        { "id": "o_m2", "name": "confirmPayment(ref)", "returnType": "PaymentResult", "visibility": "+" },
        { "id": "o_m3", "name": "recalculateTotal()", "returnType": "void", "visibility": "-" }
      ],
      "statusText": "JPA 3.1 • Active"
    },
    {
      "id": "order-item",
      "name": "OrderItem",
      "stereotype": "«Entity, Part»",
      "package": "com.nexus.orders",
      "tableBinding": "t_order_items",
      "x": 760,
      "y": 100,
      "width": 220,
      "attributes": [
        { "id": "oi_id", "name": "id", "type": "Long", "visibility": "+", "isPk": true, "annotations": ["@Id", "@GeneratedValue"] },
        { "id": "oi_sku", "name": "sku", "type": "String", "visibility": "+", "annotations": ["@NotNull"] },
        { "id": "oi_qty", "name": "quantity", "type": "Integer", "visibility": "+", "annotations": ["@Min(1)"] },
        { "id": "oi_price", "name": "unitPrice", "type": "BigDecimal", "visibility": "+", "annotations": ["@PositiveOrZero"] }
      ],
      "methods": [{ "id": "oi_m1", "name": "getSubtotal()", "returnType": "BigDecimal", "visibility": "+" }]
    },
    {
      "id": "payment",
      "name": "Payment",
      "stereotype": "«Abstract»",
      "package": "com.nexus.orders",
      "tableBinding": "t_payments",
      "x": 410,
      "y": 380,
      "width": 250,
      "isAbstract": true,
      "attributes": [
        { "id": "p_tx", "name": "transactionId", "type": "String", "visibility": "#", "annotations": ["@NotNull"] },
        { "id": "p_amt", "name": "amount", "type": "BigDecimal", "visibility": "#", "annotations": ["@PositiveOrZero"] }
      ],
      "methods": [{ "id": "p_m1", "name": "execute()", "returnType": "PaymentStatus*", "visibility": "+", "isAbstract": true }]
    },
    {
      "id": "cc-payment",
      "name": "CreditCardPayment",
      "stereotype": "«Entity»",
      "package": "com.nexus.orders",
      "tableBinding": "t_payments_cc",
      "x": 280,
      "y": 560,
      "width": 220,
      "attributes": [
        { "id": "cc_mask", "name": "cardNumberMask", "type": "String", "visibility": "-", "annotations": ["@NotNull"] },
        { "id": "cc_auth", "name": "authCode", "type": "String", "visibility": "-", "annotations": ["@NotNull"] }
      ],
      "methods": [{ "id": "cc_m1", "name": "execute()", "returnType": "PaymentStatus", "visibility": "+" }]
    },
    {
      "id": "stripe-payment",
      "name": "StripePayment",
      "stereotype": "«Entity»",
      "package": "com.nexus.orders",
      "tableBinding": "t_payments_stripe",
      "x": 540,
      "y": 560,
      "width": 230,
      "attributes": [{ "id": "sp_intent", "name": "stripePaymentIntentId", "type": "String", "visibility": "-", "annotations": ["@NotNull"] }],
      "methods": [{ "id": "sp_m1", "name": "execute()", "returnType": "PaymentStatus", "visibility": "+" }]
    }
  ],
  "relationships": [
    { "id": "rel-cust-order", "sourceId": "customer", "targetId": "order", "type": "association", "sourceMultiplicity": "1", "targetMultiplicity": "1..*", "label": "places >" },
    { "id": "rel-order-item", "sourceId": "order", "targetId": "order-item", "type": "composition", "sourceMultiplicity": "1", "targetMultiplicity": "1..*", "label": "contains" },
    { "id": "rel-order-payment", "sourceId": "order", "targetId": "payment", "type": "aggregation", "sourceMultiplicity": "1", "targetMultiplicity": "0..1", "label": "settled_by" },
    { "id": "rel-cc-payment", "sourceId": "cc-payment", "targetId": "payment", "type": "generalization" },
    { "id": "rel-stripe-payment", "sourceId": "stripe-payment", "targetId": "payment", "type": "generalization" }
  ]
}
$json$::jsonb,
      updated_at = now()
  WHERE id = 'cccccccc-cccc-cccc-cccc-cccccccccccc'
    AND document = '{"schemaVersion":1,"id":"cccccccc-cccc-cccc-cccc-cccccccccccc","name":"Diagrama principal","classes":[],"relationships":[]}'::jsonb
  RETURNING id, document, created_by
)
INSERT INTO diagram_versions (id, diagram_id, version_number, document, created_by)
SELECT 'ffffffff-ffff-ffff-ffff-ffffffffffff', id, 2, document, created_by
FROM updated_diagram;
