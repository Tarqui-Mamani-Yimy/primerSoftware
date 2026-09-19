import 'package:flutter_test/flutter_test.dart';
import 'package:ai_uml_architect_mobile/models.dart';

void main() {
  test('UML document JSON roundtrip preserves canonical fields', () {
    final original = UmlDocument(
      id: 'diagram-1',
      name: 'Orders',
      classes: [UmlClass(id: 'order', name: 'Order', attributes: [const UmlAttribute(id: 'id', name: 'id', type: 'Long', isPk: true)])],
      relationships: [const UmlRelationship(id: 'r1', sourceId: 'order', targetId: 'customer', type: 'association', targetMultiplicity: '1')],
    );
    final restored = UmlDocument.fromJson(original.toJson());
    expect(restored.toJson(), equals(original.toJson()));
  });
}
