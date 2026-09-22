class ManualDialogToken {
  const ManualDialogToken({
    required this.diagramId,
    required this.version,
    required this.reviewNumber,
    required this.generation,
  });

  final String? diagramId;
  final int version;
  final int reviewNumber;
  final int generation;

  bool matches({
    required String? diagramId,
    required int version,
    required int reviewNumber,
    required int generation,
  }) =>
      this.diagramId == diagramId &&
      this.version == version &&
      this.reviewNumber == reviewNumber &&
      this.generation == generation;
}
