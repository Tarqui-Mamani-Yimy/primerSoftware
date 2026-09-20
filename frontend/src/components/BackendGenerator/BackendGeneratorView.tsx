import React, { useState } from 'react';
import { UMLDiagramDocument } from '../../types';
import { ApiError, ArtifactConfig, artifactApi, BackendArtifactResult } from '../../api/diagramApi';
import JSZip from 'jszip';
import { es } from '../../i18n/es';

interface BackendGeneratorViewProps {
  projectId?: string;
  diagramId?: string;
  document: UMLDiagramDocument;
}

interface GeneratedArtifact {
  artifact: BackendArtifactResult;
  manifest: ArtifactManifest | null;
}

interface ArtifactManifest {
  generator?: { name?: string; version?: string; invocation?: string };
  generatedAt?: string;
  baseName?: string;
  packageName?: string;
  entities?: string[];
  relationships?: number | string[];
  warnings?: string[];
  skipped?: string[];
  files?: string[];
  fileCount?: number;
}

type GenerationPhase = 'idle' | 'generating' | 'success' | 'error';

const sanitizeGenerationError = (error: unknown): string => {
  if (!(error instanceof ApiError)) return es.generator.generationUnavailable;

  if (error.status === 400) return es.generator.generationInvalidRequest;
  if (error.status === 401) return es.generator.generationUnauthorized;
  if (error.status === 403) return es.generator.generationForbidden;
  if (error.status === 404) return es.generator.generationNotFound;
  if (error.status === 409) return es.generator.generationConflict;
  if (error.status === 429) return es.generator.generationRateLimited;
  return es.generator.generationUnavailable;
};

export const BackendGeneratorView: React.FC<BackendGeneratorViewProps> = ({
  projectId,
  diagramId,
  document
}) => {
  const [baseName, setBaseName] = useState('UmlArchitect');
  const [packageName, setPackageName] = useState('com.umlarchitect');
  const [buildTool, setBuildTool] = useState<'maven' | 'gradle'>('maven');
  const [authenticationType] = useState<'jwt'>('jwt');
  const [phase, setPhase] = useState<GenerationPhase>('idle');
  const [errorMessage, setErrorMessage] = useState('');
  const [result, setResult] = useState<GeneratedArtifact | null>(null);

  const canGenerate = Boolean(projectId && diagramId);

  const triggerDownload = (r: BackendArtifactResult) => {
    const url = window.URL.createObjectURL(r.blob);
    const link = window.document.createElement('a');
    link.href = url;
    link.download = r.filename;
    window.document.body.appendChild(link);
    link.click();
    window.document.body.removeChild(link);
    window.URL.revokeObjectURL(url);
  };

  // The server zips <baseName>/ + manifest.json; surface the manifest back to
  // the user so the board reflects real generator output instead of guesses.
  const readManifest = async (blob: Blob): Promise<ArtifactManifest | null> => {
    try {
      const zip = await JSZip.loadAsync(blob);
      const file = zip.file('manifest.json');
      if (!file) return null;
      return JSON.parse(await file.async('string')) as ArtifactManifest;
    } catch {
      return null;
    }
  };

  const handleGenerate = async () => {
    if (!projectId || !diagramId) return;
    setPhase('generating');
    setErrorMessage('');
    setResult(null);
    const config: ArtifactConfig = { baseName, packageName, buildTool, authenticationType };
    try {
      const artifact = await artifactApi.generate(projectId, diagramId, document, config);
      const manifest = await readManifest(artifact.blob);
      setResult({ artifact, manifest });
      setPhase('success');
    } catch (err) {
      setPhase('error');
      setErrorMessage(sanitizeGenerationError(err));
    }
  };

  const manifest = result?.manifest;
  const canDownload = phase === 'success' && result !== null;
  const entities = manifest?.entities ?? [];
  const warnings = manifest?.warnings ?? [];
  const skipped = manifest?.skipped ?? [];

  return (
    <div className="flex flex-col w-full min-h-[calc(100vh-4rem)] bg-[#0f131c]">
      {/* Top strip: identity + regeneration CTA */}
      <div className="px-4 py-3 bg-[#1c2028] border-b border-[#3c4a42] flex flex-wrap items-center justify-between gap-2">
        <div className="flex items-center gap-2">
          <span className="material-symbols-outlined text-sm text-[#4cd7f6]">terminal</span>
          <span className="font-heading text-sm text-[#dfe2ee] uppercase tracking-tight font-bold">{es.generator.backendTitle}</span>
        </div>
        <div className="flex flex-wrap items-center gap-1.5">
          <span className="text-[10px] px-2 py-1 bg-[#0a0e16] text-[#4edea3] font-bold flex items-center gap-1 border border-[#3c4a42]">
            <span className="material-symbols-outlined text-xs">verified</span> JHIPSTER 9.4.0
          </span>
          <span className="text-[10px] px-2 py-1 bg-[#0a0e16] text-[#4cd7f6] font-bold flex items-center gap-1 border border-[#3c4a42]">
            <span className="material-symbols-outlined text-xs">bolt</span> SPRING BOOT 4
          </span>
          <span className="text-[10px] px-2 py-1 bg-[#0a0e16] text-[#d0bcff] font-bold flex items-center gap-1 border border-[#3c4a42]">
            <span className="material-symbols-outlined text-xs">dataset</span> POSTGRESQL
          </span>
          <span className="text-[10px] px-2 py-1 bg-[#0a0e16] text-[#bbcabf] border border-[#3c4a42]">
            {buildTool.toUpperCase()} · JWT
          </span>
          <button
            onClick={() => { void handleGenerate(); }}
            disabled={!canGenerate || phase === 'generating'}
            aria-busy={phase === 'generating'}
            className="flex items-center gap-1.5 px-3.5 py-1.5 bg-[#4edea3] text-black font-heading text-xs font-bold uppercase tracking-wider hover:brightness-110 active:scale-95 transition-all shadow-md disabled:opacity-40 disabled:cursor-not-allowed disabled:active:scale-100"
            title={!canGenerate ? es.generator.noDiagram : undefined}
          >
            <span className="material-symbols-outlined text-sm">
              {phase === 'generating' ? 'hourglass_top' : 'archive'}
            </span>
            <span>{phase === 'generating' ? es.generator.generating : es.generator.generateCta}</span>
          </button>
          <button
            onClick={() => result && triggerDownload(result.artifact)}
            disabled={!canDownload}
            aria-describedby={!canDownload ? 'download-backend-help' : undefined}
            className="flex items-center gap-1.5 px-3.5 py-1.5 bg-[#4cd7f6] text-black font-heading text-xs font-bold uppercase tracking-wider hover:brightness-110 active:scale-95 transition-all shadow-md disabled:opacity-40 disabled:cursor-not-allowed disabled:active:scale-100"
          >
            <span className="material-symbols-outlined text-sm">download</span>
            <span>{es.generator.downloadCta}</span>
          </button>
          <span id="download-backend-help" className="sr-only">
            {canDownload ? es.generator.downloadReady : es.generator.downloadDisabledHint}
          </span>
        </div>
      </div>

      <div className="grid grid-cols-12 w-full flex-1 bg-[#0a0e16]">
        {/* Config column */}
        <section className="col-span-12 lg:col-span-3 bg-[#181c24] border-r border-[#3c4a42] p-3 space-y-3 font-mono text-xs">
          <div className="text-[10px] text-[#bbcabf] uppercase tracking-wider font-bold flex items-center gap-1.5">
            <span className="material-symbols-outlined text-xs text-[#4cd7f6]">tune</span>
            {es.generator.configTitle}
          </div>
          <p className="text-[10px] text-[#86948a] leading-relaxed">{es.generator.configHint}</p>

          <label className="block">
            <span className="text-[10px] text-[#bbcabf] block mb-1">{es.generator.baseName}</span>
            <input
              value={baseName}
              onChange={(e) => setBaseName(e.target.value)}
              disabled={phase === 'generating'}
              className="w-full bg-[#0a0e16] border border-[#3c4a42] px-2 py-1.5 text-[#4edea3] focus:outline-none focus:border-[#4edea3] disabled:opacity-40"
            />
          </label>

          <label className="block">
            <span className="text-[10px] text-[#bbcabf] block mb-1">{es.generator.packageName}</span>
            <input
              value={packageName}
              onChange={(e) => setPackageName(e.target.value)}
              disabled={phase === 'generating'}
              className="w-full bg-[#0a0e16] border border-[#3c4a42] px-2 py-1.5 text-[#4edea3] focus:outline-none focus:border-[#4edea3] disabled:opacity-40"
            />
          </label>

          <div>
            <span className="text-[10px] text-[#bbcabf] block mb-1">{es.generator.buildTool}</span>
            <div className="grid grid-cols-2 gap-1">
              {(['maven', 'gradle'] as const).map(tool => (
                <button
                  key={tool}
                  onClick={() => setBuildTool(tool)}
                  disabled={phase === 'generating'}
                  className={`px-1 py-1 text-center font-bold transition-colors disabled:opacity-40 ${
                    buildTool === tool
                      ? 'bg-[#4edea3] text-black shadow-sm'
                      : 'bg-[#262a33] text-[#bbcabf] hover:text-[#dfe2ee]'
                  }`}
                >
                  {tool}
                </button>
              ))}
            </div>
          </div>

          <div>
            <span className="text-[10px] text-[#bbcabf] block mb-1">{es.generator.authenticationType}</span>
            <div className="flex items-center gap-1.5 px-2 py-1.5 bg-[#0a0e16] border border-[#3c4a42] text-[#4edea3] font-bold">
              <span className="material-symbols-outlined text-xs">lock</span> JWT
            </div>
          </div>
        </section>

        {/* Result column */}
        <main className="col-span-12 lg:col-span-9 p-4 flex flex-col gap-3">
          {!canGenerate && (
            <div className="flex items-start gap-2 p-3 bg-[#262a33] border border-[#3c4a42] text-[#bbcabf] font-mono text-xs">
              <span className="material-symbols-outlined text-sm text-[#d0bcff]">warning</span>
              <span>{es.generator.noDiagram}</span>
            </div>
          )}

          {phase === 'idle' && (
            <div className="flex flex-col items-center justify-center gap-2 p-10 text-center border border-dashed border-[#3c4a42] bg-[#181c24] font-mono text-xs">
              <span className="material-symbols-outlined text-3xl text-[#4cd7f6]">terminal</span>
              <p className="text-[#dfe2ee] max-w-md leading-relaxed">{es.generator.idleHint}</p>
            </div>
          )}

          {phase === 'generating' && (
            <div role="status" aria-live="polite" className="flex flex-col items-center justify-center gap-3 p-10 text-center border border-[#3c4a42] bg-[#181c24] font-mono text-xs">
              <span className="material-symbols-outlined text-3xl text-[#4edea3] animate-pulse">hourglass_top</span>
              <p className="text-[#4edea3] font-bold">{es.generator.generating}</p>
              <p className="text-[#86948a]">{es.generator.generatingNote}</p>
            </div>
          )}

          {phase === 'error' && (
            <div role="alert" className="p-3 bg-[#2a1418] border border-[#7f2b33] font-mono text-xs">
              <div className="flex items-center gap-1.5 text-[#ff7b85] font-bold mb-1">
                <span className="material-symbols-outlined text-sm">error</span>
                {es.generator.generationFailed}
              </div>
              <pre className="whitespace-pre-wrap text-[#dfe2ee]">{errorMessage}</pre>
            </div>
          )}

          {phase === 'success' && result && (
            <div className="flex flex-col gap-3">
              <div className="flex items-start gap-2 p-3 bg-[#12241b] border border-[#2f6b45] font-mono text-xs">
                <span className="material-symbols-outlined text-sm text-[#4edea3]">download_done</span>
                <div>
                  <div className="text-[#4edea3] font-bold">{es.generator.generationSucceeded}</div>
                  <div className="text-[#bbcabf] break-all">{result.artifact.filename}</div>
                </div>
              </div>

              {manifest ? (
                <div className="p-3 bg-[#181c24] border border-[#3c4a42] font-mono text-xs space-y-3">
                  <div className="flex flex-wrap items-center gap-1.5">
                    <span className="text-[#dfe2ee] font-bold">{es.generator.manifestTitle}</span>
                    {manifest.generator?.name && manifest.generator?.version && (
                      <span className="text-[10px] px-1.5 py-0.5 bg-[#0a0e16] text-[#4cd7f6] font-bold border border-[#3c4a42]">
                        {manifest.generator.name} {manifest.generator.version}
                      </span>
                    )}
                  </div>

                  <div className="flex flex-wrap gap-4 text-[10px]">
                    <span className="text-[#bbcabf]">
                      {es.generator.entities}: <span className="text-[#4edea3] font-bold">{manifest.entities?.length ?? 0}</span>
                    </span>
                    <span className="text-[#bbcabf]">
                      {es.generator.relationships}:{' '}
                      <span className="text-[#4cd7f6] font-bold">
                        {typeof manifest.relationships === 'number' ? manifest.relationships : (manifest.relationships ?? []).length}
                      </span>
                    </span>
                    <span className="text-[#bbcabf]">
                      {es.generator.files}: <span className="text-[#dfe2ee] font-bold">{manifest.files?.length ?? manifest.fileCount ?? 0}</span>
                    </span>
                  </div>

                  {entities.length > 0 && (
                    <div className="flex flex-wrap gap-1">
                      {entities.map(name => (
                        <span key={name} className="text-[10px] px-1.5 py-0.5 bg-[#0a0e16] text-[#4edea3] border border-[#3c4a42]">{name}</span>
                      ))}
                    </div>
                  )}

                  {warnings.length > 0 && (
                    <div className="space-y-1">
                      <div className="text-[#d0bcff] font-bold">{es.generator.warnings}</div>
                      <ul className="space-y-0.5 text-[#bbcabf]">
                        {warnings.map((w, i) => <li key={i} className="flex gap-1"><span className="text-[#d0bcff]">⚠</span><span>{w}</span></li>)}
                      </ul>
                    </div>
                  )}

                  {skipped.length > 0 && (
                    <div className="space-y-1">
                      <div className="text-[#86948a] font-bold">{es.generator.skipped}</div>
                      <ul className="space-y-0.5 text-[#86948a]">
                        {skipped.map((s, i) => <li key={i}>— {s}</li>)}
                      </ul>
                    </div>
                  )}
                </div>
              ) : (
                <p className="p-3 bg-[#262a33] border border-[#3c4a42] text-[#bbcabf] font-mono text-xs">
                  {es.generator.manifestUnavailable}
                </p>
              )}
            </div>
          )}
        </main>
      </div>
    </div>
  );
};