import React, { useState } from 'react';
import { UMLClassNode, JpaStrategy, CodeFile } from '../../types';
import { generateAllCodeFiles } from '../../data/codeGenerator';
import JSZip from 'jszip';
import { es } from '../../i18n/es';

interface BackendGeneratorViewProps {
  classes: UMLClassNode[];
  strategy: JpaStrategy;
  onStrategyChange: (strategy: JpaStrategy) => void;
  onDownloadZip: () => void;
}

export const BackendGeneratorView: React.FC<BackendGeneratorViewProps> = ({
  classes,
  strategy,
  onStrategyChange
}) => {
  const codeFiles = generateAllCodeFiles(classes, strategy);
  const [activeTabId, setActiveTabId] = useState<string>('order-java');
  const [copied, setCopied] = useState(false);
  const [syncing, setSyncing] = useState(false);
  const [downloading, setDownloading] = useState(false);
  const [showDdlModal, setShowDdlModal] = useState(false);

  // Additional settings
  const [useLombok, setUseLombok] = useState(true);
  const [useUUID, setUseUUID] = useState(true);

  const activeFile = codeFiles.find(f => f.id === activeTabId) || codeFiles[0];

  const handleCopy = () => {
    navigator.clipboard.writeText(activeFile.content);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  const handleSync = () => {
    setSyncing(true);
    setTimeout(() => setSyncing(false), 1200);
  };

  const handleDownloadZip = async () => {
    setDownloading(true);
    try {
      const zip = new JSZip();
      
      // Add all generated files to zip
      codeFiles.forEach(file => {
        zip.file(file.path, file.content);
      });

      // Add README and Maven wrapper scripts
      zip.file('README.md', `# Spring Boot E-Commerce Core\nGenerated with AI UML v2.4\n\n## Build & Run\n\`\`\`bash\nmvn clean spring-boot:run\n\`\`\`\n`);
      zip.file('mvnw', '#!/bin/sh\nexec mvn "$@"\n', { unixPermissions: '755' });

      const content = await zip.generateAsync({ type: 'blob' });
      const url = window.URL.createObjectURL(content);
      const link = document.createElement('a');
      link.href = url;
      link.download = 'spring-boot-ecommerce-core.zip';
      document.body.appendChild(link);
      link.click();
      document.body.removeChild(link);
      window.URL.revokeObjectURL(url);
    } catch (err) {
      console.error('Failed to create zip', err);
    } finally {
      setDownloading(false);
    }
  };

  // Syntax highlighting helper for code view
  const renderHighlightedCode = (content: string, language: string) => {
    const lines = content.split('\n');

    return (
      <div className="grid grid-cols-12 gap-3 font-mono text-xs leading-relaxed">
        {/* Line Numbers */}
        <div className="col-span-1 text-right text-[#86948a] select-none opacity-50 space-y-0.5">
          {lines.map((_, i) => (
            <div key={i}>{String(i + 1).padStart(2, '0')}</div>
          ))}
        </div>

        {/* Syntax Lines */}
        <div className="col-span-11 space-y-0.5 overflow-x-auto text-[#dfe2ee]">
          {lines.map((line, idx) => {
            if (!line.trim()) return <div key={idx} className="h-4"></div>;

            // Comment line
            if (line.trim().startsWith('//') || line.trim().startsWith('--')) {
              return (
                <div key={idx} className="text-[#86948a] italic">
                  {line}
                </div>
              );
            }

            // Annotation line
            if (line.trim().startsWith('@')) {
              return (
                <div key={idx}>
                  <span className="text-[#d0bcff]">{line}</span>
                </div>
              );
            }

            // Java syntax coloring
            if (language === 'java') {
              const formatted = line
                .replace(/\b(package|import|public|class|private|protected|extends|implements|return|abstract|new|final)\b/g, '§CYAN$1§END')
                .replace(/\b(UUID|Instant|BigDecimal|String|Integer|Long|List|ArrayList|OrderStatus|PaymentResult|Customer|OrderItem|Payment|Order|ResponseEntity|HttpStatus|OrderResponseDTO|OrderRequestDTO|OrderService)\b/g, '§EMERALD$1§END')
                .replace(/(".*?")/g, '§GREEN$1§END');

              const parts = formatted.split(/(§CYAN.*?§END|§EMERALD.*?§END|§GREEN.*?§END)/g);

              return (
                <div key={idx} className="whitespace-pre">
                  {parts.map((part, pIdx) => {
                    if (part.startsWith('§CYAN')) {
                      return <span key={pIdx} className="text-[#4cd7f6]">{part.replace(/§CYAN|§END/g, '')}</span>;
                    }
                    if (part.startsWith('§EMERALD')) {
                      return <span key={pIdx} className="text-[#4edea3] font-bold">{part.replace(/§EMERALD|§END/g, '')}</span>;
                    }
                    if (part.startsWith('§GREEN')) {
                      return <span key={pIdx} className="text-[#4edea3]">{part.replace(/§GREEN|§END/g, '')}</span>;
                    }
                    return <span key={pIdx}>{part}</span>;
                  })}
                </div>
              );
            }

            // SQL syntax coloring
            if (language === 'sql') {
              const formatted = line
                .replace(/\b(CREATE|TABLE|EXTENSION|IF NOT EXISTS|PRIMARY KEY|DEFAULT|NOT NULL|UNIQUE|REFERENCES|ON DELETE|RESTRICT|CASCADE|INDEX|ON|TIMESTAMPTZ|NOW|NUMERIC|CHECK|INT|VARCHAR|UUID|DESC)\b/g, '§CYAN$1§END')
                .replace(/\b(t_customers|t_orders|t_order_items|t_payments|t_payments_cc|t_payments_stripe)\b/g, '§EMERALD$1§END')
                .replace(/(".*?"|'.*?')/g, '§GREEN$1§END');

              const parts = formatted.split(/(§CYAN.*?§END|§EMERALD.*?§END|§GREEN.*?§END)/g);

              return (
                <div key={idx} className="whitespace-pre">
                  {parts.map((part, pIdx) => {
                    if (part.startsWith('§CYAN')) {
                      return <span key={pIdx} className="text-[#4cd7f6]">{part.replace(/§CYAN|§END/g, '')}</span>;
                    }
                    if (part.startsWith('§EMERALD')) {
                      return <span key={pIdx} className="text-[#4edea3] font-bold">{part.replace(/§EMERALD|§END/g, '')}</span>;
                    }
                    if (part.startsWith('§GREEN')) {
                      return <span key={pIdx} className="text-[#4edea3]">{part.replace(/§GREEN|§END/g, '')}</span>;
                    }
                    return <span key={pIdx}>{part}</span>;
                  })}
                </div>
              );
            }

            return <div key={idx} className="whitespace-pre">{line}</div>;
          })}
        </div>
      </div>
    );
  };

  return (
    <div className="flex flex-col w-full min-h-[calc(100vh-4rem)] bg-[#0f131c]">
      {/* Sub-Header Breadcrumb Bar */}
      <div className="h-8 px-4 bg-[#181c24] border-b border-[#3c4a42] flex items-center gap-2 text-[#bbcabf] font-mono text-xs">
        <span className="text-[#86948a] hover:text-[#dfe2ee] cursor-pointer">ecommerce-v2</span>
        <span className="material-symbols-outlined text-xs">chevron_right</span>
        <span className="text-[#86948a] hover:text-[#dfe2ee] cursor-pointer">models</span>
        <span className="material-symbols-outlined text-xs">chevron_right</span>
        <span className="text-[#4edea3] font-bold">order_lifecycle.uml</span>
      </div>

      {/* Micro Telemetry Stream Header */}
      <div className="w-full bg-[#0a0e16] px-4 py-2 flex flex-wrap items-center justify-between gap-2 border-b border-[#3c4a42] font-mono">
        <div className="flex items-center gap-3">
          <div className="flex items-center gap-1.5">
            <span className="inline-block w-2 h-2 rounded-full bg-[#4edea3] animate-pulse"></span>
            <span className="text-[10px] text-[#4edea3] tracking-widest uppercase font-bold">{es.generator.synced}</span>
          </div>
          <span className="text-xs text-[#bbcabf]">
            Model: <span className="text-[#4cd7f6]">E-Commerce Domain (order_lifecycle.uml)</span>
          </span>
          <span className="text-[10px] px-1.5 py-0.5 bg-[#262a33] text-[#bbcabf]">HASH: e89f2a9c</span>
        </div>

        <div className="flex items-center gap-4">
          <div className="flex items-center gap-1.5">
            <span className="text-[10px] text-[#bbcabf] uppercase font-bold">{es.generator.targetStack}</span>
            <span className="text-[10px] px-2 py-0.5 bg-[#262a33] text-[#4edea3] font-bold">Java 21</span>
            <span className="text-[10px] px-2 py-0.5 bg-[#262a33] text-[#4cd7f6] font-bold">Spring Boot 3.2</span>
            <span className="text-[10px] px-2 py-0.5 bg-[#262a33] text-[#d0bcff] font-bold">PostgreSQL 16</span>
          </div>
          <div className="flex items-center gap-1.5">
            <span className="text-[10px] text-[#bbcabf] uppercase font-bold">{es.generator.buildSystem}</span>
            <span className="text-[10px] px-1.5 py-0.5 bg-[#10b981] text-[#00422b] font-bold">Maven v3.9</span>
          </div>
        </div>
      </div>

      {/* Main Split-Screen Architecture Cockpit */}
      <div className="grid grid-cols-12 w-full flex-1 bg-[#0a0e16]">
        {/* Left Panel: Project Explorer & Schema Artifacts */}
        <section className="col-span-12 lg:col-span-3 xl:col-span-3 bg-[#181c24] border-r border-[#3c4a42] flex flex-col justify-between">
          <div className="flex flex-col">
            {/* Explorer Header */}
            <div className="px-4 py-3 bg-[#1c2028] border-b border-[#3c4a42] flex items-center justify-between">
              <div className="flex items-center gap-2">
                <span className="material-symbols-outlined text-sm text-[#4cd7f6]">account_tree</span>
                <span className="font-heading text-sm text-[#dfe2ee] uppercase tracking-tight font-bold">{es.generator.artifactTree}</span>
              </div>
              <span className="text-[10px] px-1.5 py-0.5 bg-[#31353e] text-[#4cd7f6] font-mono font-bold">24 Files</span>
            </div>

            {/* Strategy Selector Control */}
            <div className="p-2.5 bg-[#0a0e16] border-b border-[#3c4a42]">
              <div className="p-2.5 bg-[#1c2028] border border-[#3c4a42]">
                <div className="flex items-center justify-between mb-1.5 font-mono">
                  <span className="text-[10px] text-[#bbcabf] uppercase tracking-wider font-bold">{es.generator.inheritanceStrategy}</span>
                  <span className="material-symbols-outlined text-xs text-[#4edea3] cursor-pointer" title="Inferred from polymorphic UML nodes (Payment -> CreditCardPayment, StripePayment)">
                    info
                  </span>
                </div>

                <div className="grid grid-cols-3 gap-1 font-mono text-xs">
                  {(['JOINED', 'SINGLE', 'TABLE_PER'] as const).map(s => (
                    <button
                      key={s}
                      onClick={() => onStrategyChange(s)}
                      className={`px-1 py-1 text-center font-bold transition-colors ${
                        strategy === s
                          ? 'bg-[#4edea3] text-black shadow-sm'
                          : 'bg-[#262a33] text-[#bbcabf] hover:text-[#dfe2ee]'
                      }`}
                    >
                      {s}
                    </button>
                  ))}
                </div>
              </div>
            </div>

            {/* File Tree */}
            <div className="p-2.5 overflow-y-auto max-h-[calc(100vh-22rem)] space-y-1 font-mono text-xs">
              {/* Root Module */}
              <div className="flex items-center gap-1.5 px-1.5 py-1 text-[#dfe2ee] font-bold bg-[#262a33]/60">
                <span className="material-symbols-outlined text-xs text-[#4edea3]">folder_open</span>
                <span className="truncate">spring-boot-ecommerce-core</span>
              </div>

              {/* Java Src Root */}
              <div className="pl-3 space-y-1">
                <div className="flex items-center gap-1 px-1 py-0.5 text-[#bbcabf]">
                  <span className="material-symbols-outlined text-xs text-[#4cd7f6]">folder_open</span>
                  <span>src/main/java/</span>
                </div>

                {/* Package */}
                <div className="pl-3 space-y-1">
                  <div className="flex items-center gap-1 px-1 py-0.5 text-[#4cd7f6]">
                    <span className="material-symbols-outlined text-xs">folder_open</span>
                    <span className="truncate">com.architect.domain</span>
                  </div>

                  {/* Model Group */}
                  <div className="pl-3 space-y-1">
                    <div className="flex items-center gap-1 px-1 py-0.5 text-[#86948a] font-semibold">
                      <span className="material-symbols-outlined text-xs text-[#d0bcff]">folder</span>
                      <span>model/</span>
                    </div>

                    <div className="pl-3 space-y-1">
                      <div
                        onClick={() => setActiveTabId('order-java')}
                        className={`cursor-pointer flex items-center justify-between px-2 py-1 transition-colors ${
                          activeTabId === 'order-java'
                            ? 'bg-[#1c2028] text-[#4edea3] font-bold border-l-2 border-[#4edea3]'
                            : 'text-[#bbcabf] hover:bg-[#1c2028] hover:text-[#dfe2ee]'
                        }`}
                      >
                        <div className="flex items-center gap-1.5 truncate">
                          <span className="material-symbols-outlined text-xs text-[#4edea3]">code</span>
                          <span>Order.java</span>
                        </div>
                        <span className="text-[9px] px-1 bg-[#10b981] text-[#00422b] font-bold">@Entity</span>
                      </div>

                      <div
                        onClick={() => setActiveTabId('customer-java')}
                        className={`cursor-pointer flex items-center justify-between px-2 py-1 transition-colors ${
                          activeTabId === 'customer-java'
                            ? 'bg-[#1c2028] text-[#4edea3] font-bold border-l-2 border-[#4edea3]'
                            : 'text-[#bbcabf] hover:bg-[#1c2028] hover:text-[#dfe2ee]'
                        }`}
                      >
                        <div className="flex items-center gap-1.5 truncate">
                          <span className="material-symbols-outlined text-xs text-[#86948a]">code</span>
                          <span>Customer.java</span>
                        </div>
                        <span className="text-[9px] text-[#86948a]">Entity</span>
                      </div>

                      <div
                        onClick={() => setActiveTabId('order-item-java')}
                        className={`cursor-pointer flex items-center justify-between px-2 py-1 transition-colors ${
                          activeTabId === 'order-item-java'
                            ? 'bg-[#1c2028] text-[#4edea3] font-bold border-l-2 border-[#4edea3]'
                            : 'text-[#bbcabf] hover:bg-[#1c2028] hover:text-[#dfe2ee]'
                        }`}
                      >
                        <div className="flex items-center gap-1.5 truncate">
                          <span className="material-symbols-outlined text-xs text-[#86948a]">code</span>
                          <span>OrderItem.java</span>
                        </div>
                        <span className="text-[9px] text-[#86948a]">Entity</span>
                      </div>

                      <div
                        onClick={() => setActiveTabId('payment-java')}
                        className={`cursor-pointer flex items-center justify-between px-2 py-1 transition-colors ${
                          activeTabId === 'payment-java'
                            ? 'bg-[#1c2028] text-[#4edea3] font-bold border-l-2 border-[#4edea3]'
                            : 'text-[#bbcabf] hover:bg-[#1c2028] hover:text-[#dfe2ee]'
                        }`}
                      >
                        <div className="flex items-center gap-1.5 truncate">
                          <span className="material-symbols-outlined text-xs text-[#86948a]">code</span>
                          <span>Payment.java</span>
                        </div>
                        <span className="text-[9px] text-[#86948a]">Entity</span>
                      </div>
                    </div>

                    {/* Repository Group */}
                    <div className="flex items-center gap-1 px-1 py-0.5 text-[#86948a] font-semibold">
                      <span className="material-symbols-outlined text-xs text-[#d0bcff]">folder</span>
                      <span>repository/</span>
                    </div>
                    <div className="pl-3 space-y-1">
                      <div className="flex items-center justify-between px-2 py-0.5 text-[#bbcabf]">
                        <div className="flex items-center gap-1.5 truncate">
                          <span className="material-symbols-outlined text-xs text-[#86948a]">terminal</span>
                          <span>OrderRepository.java</span>
                        </div>
                        <span className="text-[9px] text-[#4cd7f6]">JPA</span>
                      </div>
                      <div className="flex items-center justify-between px-2 py-0.5 text-[#bbcabf]">
                        <div className="flex items-center gap-1.5 truncate">
                          <span className="material-symbols-outlined text-xs text-[#86948a]">terminal</span>
                          <span>CustomerRepository.java</span>
                        </div>
                      </div>
                    </div>

                    {/* Controller Group */}
                    <div className="flex items-center gap-1 px-1 py-0.5 text-[#86948a] font-semibold">
                      <span className="material-symbols-outlined text-xs text-[#d0bcff]">folder</span>
                      <span>controller/</span>
                    </div>
                    <div className="pl-3 space-y-1">
                      <div
                        onClick={() => setActiveTabId('order-controller-java')}
                        className={`cursor-pointer flex items-center justify-between px-2 py-1 transition-colors ${
                          activeTabId === 'order-controller-java'
                            ? 'bg-[#1c2028] text-[#4edea3] font-bold border-l-2 border-[#4edea3]'
                            : 'text-[#bbcabf] hover:bg-[#1c2028] hover:text-[#dfe2ee]'
                        }`}
                      >
                        <div className="flex items-center gap-1.5 truncate">
                          <span className="material-symbols-outlined text-xs text-[#4cd7f6]">api</span>
                          <span>OrderRestController.java</span>
                        </div>
                        <span className="text-[9px] text-[#4cd7f6]">REST</span>
                      </div>
                    </div>
                  </div>
                </div>

                {/* Resources Root */}
                <div className="flex items-center gap-1 px-1 py-0.5 text-[#bbcabf]">
                  <span className="material-symbols-outlined text-xs text-[#d0bcff]">folder_open</span>
                  <span>src/main/resources/</span>
                </div>
                <div className="pl-3 space-y-1">
                  <div className="flex items-center gap-1 px-1 py-0.5 text-[#86948a] font-semibold">
                    <span className="material-symbols-outlined text-xs text-[#4cd7f6]">folder</span>
                    <span>db/migration/</span>
                  </div>
                  <div className="pl-3">
                    <div
                      onClick={() => setActiveTabId('sql-ddl')}
                      className={`cursor-pointer flex items-center justify-between px-2 py-1 transition-colors ${
                        activeTabId === 'sql-ddl'
                          ? 'bg-[#1c2028] text-[#4edea3] font-bold border-l-2 border-[#4edea3]'
                          : 'text-[#bbcabf] hover:bg-[#1c2028] hover:text-[#dfe2ee]'
                      }`}
                    >
                      <div className="flex items-center gap-1.5 truncate">
                        <span className="material-symbols-outlined text-xs text-[#d0bcff]">database</span>
                        <span>V1__init_schema.sql</span>
                      </div>
                      <span className="text-[9px] px-1.5 py-0.5 bg-[#b090ff] text-[#4600a7] font-bold">DDL</span>
                    </div>
                  </div>

                  <div
                    onClick={() => setActiveTabId('application-yml')}
                    className={`cursor-pointer flex items-center gap-1.5 px-2 py-1 transition-colors ${
                      activeTabId === 'application-yml'
                        ? 'bg-[#1c2028] text-[#4edea3] font-bold'
                        : 'text-[#bbcabf] hover:bg-[#1c2028] hover:text-[#dfe2ee]'
                    }`}
                  >
                    <span className="material-symbols-outlined text-xs text-[#86948a]">settings</span>
                    <span>application.yml</span>
                  </div>
                </div>
              </div>
            </div>
          </div>

          {/* Left Panel Bottom Quick Metric */}
          <div className="p-3 bg-[#1c2028] m-2 border border-[#3c4a42] font-mono">
            <div className="flex items-center justify-between text-[#bbcabf] mb-1.5">
              <span className="text-[10px] uppercase font-bold">Schema Fidelity</span>
              <span className="text-xs text-[#4edea3] font-bold">100%</span>
            </div>
            <div className="w-full bg-[#0a0e16] h-1.5 overflow-hidden">
              <div className="bg-[#4edea3] h-full w-full"></div>
            </div>
            <p className="text-[11px] text-[#bbcabf] mt-1.5 leading-tight">
              All 4 UML Classes, 6 Cardinalities &amp; 14 Attributes completely synchronized.
            </p>
          </div>
        </section>

        {/* Center-Right Panel: Multi-Tab High-Fidelity Code Studio */}
        <main className="col-span-12 lg:col-span-9 xl:col-span-9 flex flex-col justify-between bg-[#0a0e16]">
          <div>
            {/* Top IDE Control Bar & Quick Exports */}
            <div className="p-2.5 bg-[#1c2028] border-b border-[#3c4a42] flex flex-wrap items-center justify-between gap-2 font-mono">
              <div className="flex flex-wrap items-center gap-1.5">
                <span className="text-[10px] px-2 py-1 bg-[#0a0e16] text-[#4edea3] font-bold flex items-center gap-1 border border-[#3c4a42]">
                  <span className="material-symbols-outlined text-xs">verified</span> JAVA 21 JPA
                </span>
                <span className="text-[10px] px-2 py-1 bg-[#0a0e16] text-[#4cd7f6] font-bold flex items-center gap-1 border border-[#3c4a42]">
                  <span className="material-symbols-outlined text-xs">bolt</span> SPRING BOOT 3.2.3
                </span>
                <span className="text-[10px] px-2 py-1 bg-[#0a0e16] text-[#d0bcff] font-bold flex items-center gap-1 border border-[#3c4a42]">
                  <span className="material-symbols-outlined text-xs">dataset</span> POSTGRESQL 16
                </span>
                <span className="text-[10px] px-2 py-1 bg-[#0a0e16] text-[#bbcabf] border border-[#3c4a42]">
                  FLYWAY MIGRATIONS ON
                </span>
              </div>

              {/* CTAs */}
              <div className="flex items-center gap-2">
                <button
                  onClick={handleCopy}
                  className="flex items-center gap-1 px-3 py-1.5 bg-[#262a33] hover:bg-[#31353e] text-[#dfe2ee] text-xs font-semibold transition-colors border border-[#3c4a42]"
                >
                  <span className={`material-symbols-outlined text-xs ${copied ? 'text-[#4edea3]' : 'text-[#4cd7f6]'}`}>
                    {copied ? 'done' : 'content_copy'}
                  </span>
                  <span>{copied ? `¡${es.generator.copied}!` : es.generator.copy}</span>
                </button>

                <button
                  onClick={handleSync}
                  className="hidden md:flex items-center gap-1 px-3 py-1.5 bg-[#262a33] hover:bg-[#31353e] text-[#dfe2ee] text-xs font-semibold transition-colors border border-[#3c4a42]"
                >
                  <span className={`material-symbols-outlined text-xs text-[#4edea3] ${syncing ? 'animate-spin' : ''}`}>
                    sync
                  </span>
                  <span>{syncing ? 'Sincronizando…' : es.generator.synchronize}</span>
                </button>

                <button
                  onClick={handleDownloadZip}
                  disabled={downloading}
                  className="flex items-center gap-1.5 px-3.5 py-1.5 bg-[#4edea3] text-black font-heading text-xs font-bold uppercase tracking-wider hover:brightness-110 active:scale-95 transition-all shadow-md"
                >
                  <span className="material-symbols-outlined text-sm">
                    {downloading ? 'hourglass_top' : 'archive'}
                  </span>
                  <span>{downloading ? 'Generando ZIP...' : 'Descargar .ZIP (Maven)'}</span>
                  <span className="text-[10px] opacity-80 font-mono">(24 Files • 48KB)</span>
                </button>
              </div>
            </div>

            {/* Editor Tab Bar */}
            <div className="flex items-center bg-[#181c24] px-2 pt-1 gap-1 overflow-x-auto border-b border-[#3c4a42] font-mono text-xs">
              {[
                { id: 'order-java', name: 'Order.java', icon: 'code', iconColor: 'text-[#4edea3]' },
                { id: 'sql-ddl', name: 'V1__init_schema.sql', icon: 'database', iconColor: 'text-[#d0bcff]' },
                { id: 'order-controller-java', name: 'OrderRestController.java', icon: 'api', iconColor: 'text-[#4cd7f6]' },
                { id: 'customer-java', name: 'Customer.java', icon: 'code', iconColor: 'text-[#86948a]' }
              ].map(tab => (
                <button
                  key={tab.id}
                  onClick={() => setActiveTabId(tab.id)}
                  className={`flex items-center gap-1.5 px-3 py-1.5 transition-colors ${
                    activeTabId === tab.id
                      ? 'bg-[#0a0e16] text-[#4edea3] font-bold border-t border-x border-[#3c4a42]'
                      : 'bg-[#262a33] text-[#bbcabf] hover:text-[#dfe2ee]'
                  }`}
                >
                  <span className={`material-symbols-outlined text-xs ${tab.iconColor}`}>{tab.icon}</span>
                  <span>{tab.name}</span>
                  <span className="material-symbols-outlined text-xs text-[#bbcabf] hover:text-[#dfe2ee] ml-1">close</span>
                </button>
              ))}
            </div>

            {/* Code Viewing Region */}
            <div className="relative p-4 bg-[#0a0e16] overflow-x-auto min-h-[38rem]">
              {/* Breadcrumb in editor */}
              <div className="flex items-center justify-between pb-2 mb-3 text-[#bbcabf] font-mono text-xs border-b border-[#3c4a42]">
                <div className="flex items-center gap-1">
                  <span className="text-[#86948a]">src</span>
                  <span>/</span>
                  <span className="text-[#86948a]">main</span>
                  <span>/</span>
                  <span className="text-[#86948a]">{activeFile.language === 'sql' ? 'resources' : 'java'}</span>
                  <span>/</span>
                  <span className="text-[#4cd7f6]">
                    {activeFile.language === 'sql' ? 'db.migration' : 'com.architect.domain.model'}
                  </span>
                  <span>/</span>
                  <span className="text-[#4edea3] font-bold">{activeFile.filename}</span>
                </div>

                <div className="flex items-center gap-3">
                  <span className="text-[10px] text-[#bbcabf]">UTF-8</span>
                  <span className="text-[10px] text-[#bbcabf]">LF</span>
                  <span className="text-[10px] px-1.5 py-0.5 bg-[#1c2028] text-[#4edea3] font-bold border border-[#3c4a42]">
                    Generated via AST-v2
                  </span>
                </div>
              </div>

              {/* Code pane */}
              {renderHighlightedCode(activeFile.content, activeFile.language)}
            </div>
          </div>

          {/* Bottom Dock: Architecture Validation Diagnostic Feed */}
          <footer className="p-3 bg-[#1c2028] flex flex-wrap items-center justify-between gap-3 border-t border-[#3c4a42] font-mono">
            <div className="flex flex-wrap items-center gap-3">
              <div className="flex items-center gap-1.5 px-2 py-1 bg-[#0a0e16] border border-[#3c4a42]">
                <span className="material-symbols-outlined text-sm text-[#4edea3]">check_circle</span>
                <span className="text-xs text-[#4edea3] font-bold">0 COMPILATION WARNINGS</span>
              </div>
              <div className="flex items-center gap-1 text-[#bbcabf] text-xs">
                <span className="material-symbols-outlined text-xs text-[#4cd7f6]">verified_user</span>
                <span>UML Bidirectional Associations verified in JPA 3.2</span>
              </div>
              <div className="flex items-center gap-1 text-[#bbcabf] text-xs">
                <span className="material-symbols-outlined text-xs text-[#d0bcff]">key</span>
                <span>All Foreign Keys &amp; Compound Indices mapped</span>
              </div>
            </div>

            <div className="flex items-center gap-4">
              <div className="flex items-center gap-2 text-xs text-[#bbcabf]">
                <span>Memory: <span className="text-[#4cd7f6]">182MB</span></span>
                <span>|</span>
                <span>Generation Latency: <span className="text-[#4edea3]">18ms</span></span>
              </div>
              <button
                onClick={() => setShowDdlModal(true)}
                className="flex items-center gap-1 px-2 py-1 bg-[#262a33] hover:bg-[#31353e] text-[#bbcabf] hover:text-[#dfe2ee] text-[10px] uppercase border border-[#3c4a42]"
              >
                <span className="material-symbols-outlined text-xs">tune</span> DDL Settings
              </button>
            </div>
          </footer>
        </main>
      </div>

      {/* DDL Settings Modal */}
      {showDdlModal && (
        <div className="fixed inset-0 z-50 bg-black/70 backdrop-blur-sm flex items-center justify-center p-4">
          <div className="bg-[#181c24] border border-[#3c4a42] max-w-md w-full p-4 shadow-2xl font-mono text-xs">
            <div className="flex items-center justify-between pb-2 border-b border-[#3c4a42]">
              <span className="text-[#4edea3] font-bold uppercase">PostgreSQL DDL &amp; JPA Settings</span>
              <button onClick={() => setShowDdlModal(false)} className="text-[#bbcabf] hover:text-white">
                <span className="material-symbols-outlined text-sm">close</span>
              </button>
            </div>

            <div className="py-3 space-y-3">
              <div className="flex items-center justify-between">
                <span>Generate Lombok Annotations (@Getter, @Builder)</span>
                <input
                  type="checkbox"
                  checked={useLombok}
                  onChange={(e) => setUseLombok(e.target.checked)}
                  className="accent-[#4edea3]"
                />
              </div>
              <div className="flex items-center justify-between">
                <span>Use UUID PK Generation (`gen_random_uuid()`)</span>
                <input
                  type="checkbox"
                  checked={useUUID}
                  onChange={(e) => setUseUUID(e.target.checked)}
                  className="accent-[#4edea3]"
                />
              </div>
              <div>
                <label className="text-[10px] text-[#bbcabf] block mb-1">Target PostgreSQL Version</label>
                <select className="w-full bg-[#0a0e16] border border-[#3c4a42] p-1 text-[#4edea3]">
                  <option>PostgreSQL 16 (Native UUIDv7, Enhanced JSONB)</option>
                  <option>PostgreSQL 15</option>
                  <option>PostgreSQL 14</option>
                </select>
              </div>
            </div>

            <div className="flex justify-end gap-2 pt-2 border-t border-[#3c4a42]">
              <button
                onClick={() => setShowDdlModal(false)}
                className="px-3 py-1 bg-[#4edea3] text-black font-bold"
              >
                Apply Settings
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};
