import React, { useState, useEffect } from 'react';
import { ActiveView, UMLClassNode } from '../../types';

interface FloatingAssistantProps {
  activeView: ActiveView;
  selectedClassId?: string;
  classes: UMLClassNode[];
}

interface Recommendation {
  id: string;
  category: 'architecture' | 'database' | 'performance' | 'best-practice';
  title: string;
  description: string;
  actionHint?: string;
}

export const FloatingAssistant: React.FC<FloatingAssistantProps> = ({
  activeView,
  selectedClassId,
  classes
}) => {
  const [isOpen, setIsOpen] = useState(false);
  const [showNotificationBadge, setShowNotificationBadge] = useState(true);
  const [currentTipIndex, setCurrentTipIndex] = useState(0);
  const [minimizedTip, setMinimizedTip] = useState(true);
  const [userFeedback, setUserFeedback] = useState<string | null>(null);

  const selectedClass = classes.find(c => c.id === selectedClassId);

  // Dynamic recommendations depending on active view and context
  const getContextualRecommendations = (): Recommendation[] => {
    switch (activeView) {
      case 'uml-canvas':
        return [
          {
            id: 'tip-uml-1',
            category: 'architecture',
            title: `Modelando ${selectedClass?.name || 'Entidades'}`,
            description: selectedClass?.id === 'order'
              ? 'Has definido Order como Aggregate Root. Verifica que las transacciones modifiquen OrderItem únicamente a través de la raíz para respetar DDD.'
              : 'Revisa si los atributos marcados como clave foránea tienen sus relaciones bidireccionales mapeadas con cascade = ALL.',
            actionHint: 'Tip: Usa el botón Layout para alinear conectores ortogonales.'
          },
          {
            id: 'tip-uml-2',
            category: 'database',
            title: 'Índices en Claves Foráneas',
            description: 'PostgreSQL no crea índices automáticos en columnas con Foreign Key. Conviene añadir un índice en customer_id para evitar Sequential Scans.',
            actionHint: 'Se incluirá en el script de migración Flyway automáticamente.'
          },
          {
            id: 'tip-uml-3',
            category: 'best-practice',
            title: 'Atributos Monetarios con BigDecimal',
            description: 'Evita tipos flotantes (float/double) en totalAmount y unitPrice para prevenir errores de redondeo de punto flotante en transacciones bancarias.',
            actionHint: 'El generador ya usa BigDecimal(12, 2) por seguridad financiera.'
          }
        ];

      case 'backend-db-generator':
        return [
          {
            id: 'tip-gen-1',
            category: 'performance',
            title: 'Estrategia de Herencia JPA',
            description: 'Actualmente estás en el generador de backend. La estrategia JOINED crea tablas individuales limpias para Payment, CreditCardPayment y StripePayment con llave foránea compartida.',
            actionHint: 'Puedes descargar el .ZIP completo de Maven listo para compilar.'
          },
          {
            id: 'tip-gen-2',
            category: 'database',
            title: 'Versionado con Flyway',
            description: 'Observo que estás visualizando V1__init_schema.sql. Recuerda que no debes editar migraciones ya ejecutadas en producción; en su lugar crea una V2.',
            actionHint: 'El archivo descargable incluye scripts wrapper ./mvnw.'
          }
        ];

      case 'git-diff-versions':
        return [
          {
            id: 'tip-git-1',
            category: 'best-practice',
            title: 'Revisión Previa al Merge',
            description: 'Estás comparando la rama feat/payment-order-v2. Hay 6 adiciones y 2 supresiones. Comprueba que las columnas con @NotNull tengan valores por defecto para no romper datos previos.',
            actionHint: 'Todo listo para abrir Pull Request en tu repositorio Git.'
          }
        ];

      case 'team-room-live':
        return [
          {
            id: 'tip-team-1',
            category: 'architecture',
            title: 'Sesión Colaborativa Activa',
            description: 'Sarah y Alex están en la sala. Cualquier movimiento de nodos o cambio en atributos se sincroniza por WebRTC en menos de 15ms.',
            actionHint: 'Utiliza el puntero láser en el lienzo si deseas señalar una relación.'
          }
        ];

      case 'ai-architect-console':
        return [
          {
            id: 'tip-ai-1',
            category: 'architecture',
            title: 'Sugerencias de Refactorización',
            description: 'En esta consola puedes solicitar patrones avanzados como Transactional Outbox, DTOs con MapStruct o auditoría con Spring Data Envers.',
            actionHint: 'Prueba escribir: "Añadir Value Object Money / Currency".'
          }
        ];

      default:
        return [
          {
            id: 'tip-def-1',
            category: 'best-practice',
            title: 'Asistente Observando',
            description: 'Estoy analizando tu navegación y cambios arquitectónicos en tiempo real.',
            actionHint: 'Haz clic en cualquier clase para inspeccionar recomendaciones.'
          }
        ];
    }
  };

  const recommendations = getContextualRecommendations();
  const currentTip = recommendations[currentTipIndex % recommendations.length];

  // Auto-switch recommendations on view change
  useEffect(() => {
    setCurrentTipIndex(0);
    setUserFeedback(null);
  }, [activeView, selectedClassId]);

  const handleNextTip = () => {
    setCurrentTipIndex(prev => (prev + 1) % recommendations.length);
    setUserFeedback(null);
  };

  const handleHelpful = (helpful: boolean) => {
    setUserFeedback(helpful ? '¡Gracias por el feedback!' : 'Tomado en cuenta para los próximos consejos.');
    setTimeout(() => {
      setUserFeedback(null);
    }, 2500);
  };

  return (
    <aside aria-label="Asistente IA flotante" className="fixed bottom-6 right-6 z-50 flex flex-col items-end pointer-events-auto font-mono select-none">
      {/* Speech Bubble / Mini Prompt Preview when collapsed */}
      {!isOpen && minimizedTip && (
        <div 
          id="assistant-speech-bubble"
          className="mb-3 max-w-xs bg-[#181c24]/95 border border-[#3c4a42] p-3 shadow-2xl backdrop-blur-md text-xs relative animate-fade-in transition-all"
        >
          {/* Triangular tail pointing down towards the bubble */}
          <div className="absolute -bottom-2 right-6 w-3 h-3 bg-[#181c24] border-r border-b border-[#3c4a42] transform rotate-45"></div>

          <div className="flex items-start justify-between gap-2 mb-1">
            <div className="flex items-center gap-1.5 text-[10px] text-[#4edea3] font-bold">
              <span className="w-2 h-2 rounded-full bg-[#4edea3] animate-ping"></span>
              <span>Asistente Observador</span>
            </div>
            <button
              onClick={(e) => {
                e.stopPropagation();
                setMinimizedTip(false);
              }}
              className="text-[#86948a] hover:text-[#dfe2ee]"
              title="Ocultar globo de texto"
            >
              <span className="material-symbols-outlined text-xs">close</span>
            </button>
          </div>

          <p className="text-[11px] text-[#dfe2ee] leading-relaxed cursor-pointer" onClick={() => setIsOpen(true)}>
            💡 <strong className="text-[#4cd7f6]">{currentTip.title}:</strong> {currentTip.description.slice(0, 110)}...
          </p>

          <div className="mt-2 pt-1.5 border-t border-[#3c4a42] flex items-center justify-between text-[10px]">
            <span className="text-[#86948a]">👀 Mirando tu pantalla</span>
            <button
              onClick={() => setIsOpen(true)}
              className="text-[#4edea3] hover:underline font-bold flex items-center gap-0.5"
            >
              <span>Ver consejo</span>
              <span className="material-symbols-outlined text-xs">arrow_forward</span>
            </button>
          </div>
        </div>
      )}

      {/* Expanded Assistant Card Window */}
      {isOpen && (
        <div 
          id="assistant-floating-card"
          className="mb-3 w-80 sm:w-96 bg-[#181c24] border border-[#4edea3]/40 shadow-2xl shadow-black/80 flex flex-col overflow-hidden animate-fade-in"
        >
          {/* Header */}
          <div className="px-3 py-2.5 bg-[#1c2028] border-b border-[#3c4a42] flex items-center justify-between">
            <div className="flex items-center gap-2">
              <div className="relative">
                <div className="w-7 h-7 rounded-full bg-[#00422b] text-[#4edea3] border border-[#4edea3] flex items-center justify-center">
                  <span className="material-symbols-outlined text-sm">smart_toy</span>
                </div>
                <span className="absolute -bottom-0.5 -right-0.5 w-2 h-2 rounded-full bg-[#4edea3] ring-1 ring-[#181c24] animate-pulse"></span>
              </div>
              <div>
                <div className="flex items-center gap-1.5">
                  <span className="font-heading text-xs text-white font-bold tracking-wide">Asistente Guía IA</span>
                  <span className="text-[9px] px-1 bg-[#10b981]/20 text-[#4edea3] font-bold border border-[#4edea3]/30">
                    VIVO
                  </span>
                </div>
                <div className="text-[10px] text-[#86948a] flex items-center gap-1">
                  <span>Observando flujo de trabajo</span>
                </div>
              </div>
            </div>

            <div className="flex items-center gap-1">
              <button
                onClick={() => setIsOpen(false)}
                className="p-1 text-[#86948a] hover:text-white transition-colors"
                title="Minimizar a burbuja"
              >
                <span className="material-symbols-outlined text-sm">remove</span>
              </button>
              <button
                onClick={() => {
                  setIsOpen(false);
                  setMinimizedTip(false);
                }}
                className="p-1 text-[#86948a] hover:text-white transition-colors"
                title="Cerrar asistente"
              >
                <span className="material-symbols-outlined text-sm">close</span>
              </button>
            </div>
          </div>

          {/* Current Activity Detector Banner */}
          <div className="px-3 py-1.5 bg-[#0a0e16] border-b border-[#3c4a42] flex items-center justify-between text-[11px] text-[#bbcabf]">
            <div className="flex items-center gap-1.5 truncate">
              <span className="material-symbols-outlined text-xs text-[#4cd7f6]">visibility</span>
              <span className="truncate">
                Actividad: <span className="text-[#4cd7f6] font-bold">
                  {activeView === 'uml-canvas' ? 'Modelando UML Canvas' :
                   activeView === 'backend-db-generator' ? 'Generador Spring Boot & SQL' :
                   activeView === 'git-diff-versions' ? 'Comparador Git Diff' :
                   activeView === 'team-room-live' ? 'Sala WebRTC en Equipo' : 'Consola de IA'}
                </span>
              </span>
            </div>
            {selectedClass && activeView === 'uml-canvas' && (
              <span className="text-[9px] px-1 bg-[#262a33] text-[#d0bcff] font-bold">
                {selectedClass.name}
              </span>
            )}
          </div>

          {/* Assistant Recommendation Body */}
          <div className="p-3 space-y-3 bg-[#121620]">
            <div className="p-3 bg-[#1c2028] border border-[#3c4a42] space-y-2">
              <div className="flex items-center justify-between">
                <span className="text-[10px] text-[#4edea3] font-bold uppercase tracking-wider flex items-center gap-1">
                  <span className="material-symbols-outlined text-xs">lightbulb</span>
                  Recomendación ({currentTipIndex + 1}/{recommendations.length})
                </span>
                <span className="text-[9px] px-1.5 py-0.5 bg-[#0a0e16] text-[#bbcabf] uppercase font-bold">
                  {currentTip.category}
                </span>
              </div>

              <h4 className="text-xs text-white font-bold leading-snug">
                {currentTip.title}
              </h4>

              <p className="text-[11px] text-[#dfe2ee] leading-relaxed">
                {currentTip.description}
              </p>

              {currentTip.actionHint && (
                <div className="p-2 bg-[#0a0e16] border-l-2 border-[#4edea3] text-[10px] text-[#4edea3]">
                  {currentTip.actionHint}
                </div>
              )}
            </div>

            {/* Quick Interactive Actions */}
            <div className="flex items-center justify-between pt-1">
              <button
                onClick={handleNextTip}
                className="px-2.5 py-1 bg-[#262a33] hover:bg-[#31353e] text-[11px] text-[#dfe2ee] border border-[#3c4a42] flex items-center gap-1 transition-colors"
              >
                <span className="material-symbols-outlined text-xs">sync</span>
                <span>Otro consejo</span>
              </button>

              <div className="flex items-center gap-1">
                <span className="text-[10px] text-[#86948a]">¿Útil?</span>
                <button
                  onClick={() => handleHelpful(true)}
                  className="p-1 hover:bg-[#262a33] text-[#86948a] hover:text-[#4edea3]"
                  title="Útil"
                >
                  <span className="material-symbols-outlined text-xs">thumb_up</span>
                </button>
                <button
                  onClick={() => handleHelpful(false)}
                  className="p-1 hover:bg-[#262a33] text-[#86948a] hover:text-[#ffb4ab]"
                  title="No relevante"
                >
                  <span className="material-symbols-outlined text-xs">thumb_down</span>
                </button>
              </div>
            </div>

            {userFeedback && (
              <div className="text-[10px] text-[#4edea3] text-center bg-[#00422b]/30 py-1 border border-[#4edea3]/30">
                {userFeedback}
              </div>
            )}
          </div>

          {/* Bottom Status / Disclaimer */}
          <div className="px-3 py-2 bg-[#181c24] border-t border-[#3c4a42] text-[10px] text-[#86948a] flex items-center justify-between">
            <span className="flex items-center gap-1">
              <span className="w-1.5 h-1.5 rounded-full bg-[#4edea3]"></span>
              <span>Modo Observador activo</span>
            </span>
            <span className="text-[9px] text-[#86948a]">AI Companion v1.0</span>
          </div>
        </div>
      )}

      {/* FLOATING ACTION BUBBLE BUTTON */}
      <div className="relative group">
        <button
          id="floating-assistant-toggle-button"
          onClick={() => {
            setIsOpen(!isOpen);
            setShowNotificationBadge(false);
            if (!isOpen) setMinimizedTip(false);
          }}
          className={`w-13 h-13 rounded-full flex items-center justify-center transition-all duration-300 shadow-xl cursor-pointer ${
            isOpen
              ? 'bg-[#4edea3] text-black ring-4 ring-[#4edea3]/20 rotate-90 scale-105'
              : 'bg-[#181c24] hover:bg-[#1c2028] text-[#4edea3] border-2 border-[#4edea3] hover:scale-110 shadow-black/80'
          }`}
          style={{ width: '52px', height: '52px' }}
          title={isOpen ? 'Cerrar asistente' : 'Abrir asistente guía'}
          aria-expanded={isOpen}
          aria-label="Abrir asistente observador"
        >
          {isOpen ? (
            <span className="material-symbols-outlined text-2xl">close</span>
          ) : (
            <div className="relative flex items-center justify-center">
              <span className="material-symbols-outlined text-2xl">smart_toy</span>
              {/* Subtle orbital radar animation */}
              <span className="absolute inset-0 -m-1 rounded-full border border-[#4edea3] animate-ping opacity-40"></span>
            </div>
          )}
        </button>

        {/* Unread Advice Notification Badge */}
        {!isOpen && showNotificationBadge && (
          <span 
            id="assistant-notification-badge"
            className="absolute -top-1 -right-1 w-4 h-4 bg-[#ff5555] text-white text-[9px] font-bold rounded-full flex items-center justify-center ring-2 ring-[#0f131c]"
          >
            !
          </span>
        )}

        {/* Hover Tooltip when collapsed */}
        {!isOpen && !minimizedTip && (
          <div className="absolute right-full mr-3 top-1/2 -translate-y-1/2 hidden group-hover:block whitespace-nowrap bg-[#181c24] border border-[#3c4a42] px-2.5 py-1 text-xs text-[#dfe2ee] shadow-lg">
            Asistente IA Observador
          </div>
        )}
      </div>
    </aside>
  );
};
