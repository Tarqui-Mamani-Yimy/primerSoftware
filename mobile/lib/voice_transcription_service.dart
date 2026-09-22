import 'dart:async';
import 'dart:io';
import 'dart:typed_data';

import 'package:whisper_cpp_flutter_plus/whisper_cpp_flutter_plus.dart';

class VoicePermissionException implements Exception {
  const VoicePermissionException();
}

class VoiceTranscriptionService {
  static const modelName = 'ggml-tiny-q5_1.bin';
  static const modelUrl =
      'https://huggingface.co/ggerganov/whisper.cpp/resolve/'
      '5359861c739e955e79d9a303bcbc70fb988958b1/ggml-tiny-q5_1.bin';
  static const modelSha256 =
      '818710568da3ca15689e31a743197b520007872ff9576237bda97bd1b469c3d7';
  static const WhisperModelDescriptor model = WhisperModelDescriptor(
    id: 'tiny-q5_1',
    fileName: modelName,
    url: modelUrl,
    sha256: modelSha256,
    approximateBytes: 32152673,
    languageScope: WhisperModelLanguageScope.multilingual,
    purpose: WhisperModelPurpose.transcription,
  );

  /// Voice commands are a Spanish-only grammar. Auto-detection with the tiny
  /// model misclassifies short utterances and may translate them, so the
  /// language is pinned.
  static const TranscribeOptions transcribeOptions =
      TranscribeOptions(language: 'es');

  final WhisperModelManager _modelManager = WhisperModelManager();
  final WhisperRecorder _recorder = WhisperRecorder();
  final List<double> _samples = <double>[];
  WhisperEngine? _engine;
  StreamSubscription<RecordingChunk>? _recordingSubscription;

  bool get isRecording => _recordingSubscription != null;

  Future<void> prepare({void Function(double?)? onDownloadProgress}) async {
    _ensureAndroid();
    final existing = await _modelManager.findCatalogModel(model);
    final installedModel = existing ??
        await _downloadModel(onDownloadProgress: onDownloadProgress);
    _engine ??= await WhisperEngine.load(installedModel.path);
  }

  Future<File> _downloadModel(
      {void Function(double?)? onDownloadProgress}) async {
    await for (final progress in _modelManager.downloadCatalogModel(model)) {
      onDownloadProgress?.call(progress.fraction);
    }
    final downloaded = await _modelManager.findCatalogModel(model);
    if (downloaded == null) throw StateError('El modelo no quedó disponible.');
    return downloaded;
  }

  Future<void> startRecording(
      {void Function(double?)? onDownloadProgress}) async {
    await prepare(onDownloadProgress: onDownloadProgress);
    if (!await _recorder.requestPermission()) {
      throw const VoicePermissionException();
    }
    _samples.clear();
    final stream = await _recorder.start();
    _recordingSubscription =
        stream.listen((chunk) => _samples.addAll(chunk.samples));
  }

  Future<String> stopAndTranscribe() async {
    if (!isRecording) throw StateError('No hay una grabación activa.');
    await _recorder.stop();
    await _recordingSubscription?.cancel();
    _recordingSubscription = null;
    if (_samples.isEmpty) throw StateError('No se capturó audio.');
    final engine = _engine;
    if (engine == null) throw StateError('El modelo no está cargado.');
    final task = engine.transcribe(
      Float32List.fromList(_samples),
      options: transcribeOptions
          .withPerformanceMode(WhisperPerformanceMode.efficient),
    );
    final result = await task.result;
    return result.text.trim();
  }

  Future<void> dispose() async {
    await _recorder.stop();
    await _recordingSubscription?.cancel();
    _recordingSubscription = null;
    _engine?.dispose();
    _engine = null;
    _modelManager.close();
  }

  void _ensureAndroid() {
    if (!Platform.isAndroid) {
      throw UnsupportedError(
          'La transcripción de voz solo está disponible en Android.');
    }
  }
}
