import 'package:ai_uml_architect_mobile/manual_dialog_guard.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('rejects a dialog result after document generation changes', () {
    const token = ManualDialogToken(
      diagramId: 'diagram-1',
      version: 3,
      reviewNumber: 7,
      generation: 10,
    );

    expect(
      token.matches(
        diagramId: 'diagram-1',
        version: 3,
        reviewNumber: 7,
        generation: 10,
      ),
      isTrue,
    );
    expect(
      token.matches(
        diagramId: 'diagram-1',
        version: 3,
        reviewNumber: 7,
        generation: 11,
      ),
      isFalse,
    );
  });

  test('rejects stale results after diagram or CAS baseline changes', () {
    const token = ManualDialogToken(
      diagramId: 'diagram-1',
      version: 3,
      reviewNumber: 7,
      generation: 10,
    );

    expect(
      token.matches(
        diagramId: 'diagram-2',
        version: 3,
        reviewNumber: 7,
        generation: 10,
      ),
      isFalse,
    );
    expect(
      token.matches(
        diagramId: 'diagram-1',
        version: 4,
        reviewNumber: 7,
        generation: 10,
      ),
      isFalse,
    );
    expect(
      token.matches(
        diagramId: 'diagram-1',
        version: 3,
        reviewNumber: 8,
        generation: 10,
      ),
      isFalse,
    );
  });
}
