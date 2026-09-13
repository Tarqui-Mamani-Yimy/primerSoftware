import React, { useState } from 'react';

interface AiConsoleViewProps {
  onApplyRefactor?: (prompt: string) => void;
}

export const AiConsoleView: React.FC<AiConsoleViewProps> = ({ onApplyRefactor }) => {
  const [messages, setMessages] = useState([
    {
      sender: 'ai',
      text: 'Bienvenido al AI Architect Console. Estoy analizando el modelo de dominio `E-Commerce Domain Model (order_lifecycle.uml)`. He verificado 4 clases, 6 relaciones y la compatibilidad con JPA 3.2 + PostgreSQL 16. ¿Qué optimización o entidad deseas agregar?'
    }
  ]);
  const [query, setQuery] = useState('');
  const [isAnalyzing, setIsAnalyzing] = useState(false);

  const quickPrompts = [
    'Añadir Value Object Money / Currency con validaciones',
    'Optimizar índices para búsquedas de Order por created_at DESC',
    'Implementar patrón Outbox para eventos de Payment',
    'Generar DTOs con Bean Validation (@NotNull, @Positive)'
  ];

  const handleSend = (textToSend?: string) => {
    const text = textToSend || query;
    if (!text.trim()) return;

    setMessages(prev => [...prev, { sender: 'user', text }]);
    setQuery('');
    setIsAnalyzing(true);

    setTimeout(() => {
      let responseText = '';
      if (text.includes('Money') || text.includes('Currency')) {
        responseText = 'He generado el Value Object `@Embeddable public record Money(BigDecimal amount, Currency currency)`. Se puede vincular a `Order.totalAmount` y `OrderItem.unitPrice` sin alterar las llaves foráneas.';
      } else if (text.includes('índice') || text.includes('Order')) {
        responseText = 'He verificado `idx_order_created_at` en `t_orders(created_at DESC)`. Para optimizar consultas combinadas de cliente y fecha, sugiero un índice compuesto: `@Index(name = "idx_order_cust_created", columnList = "customer_id, created_at DESC")`.';
      } else if (text.includes('Outbox')) {
        responseText = 'Diseñado: Nueva entidad `OutboxEvent` (`id UUID`, `aggregate_type VARCHAR`, `aggregate_id UUID`, `payload JSONB`, `created_at TIMESTAMPTZ`). Garantiza transacciones ACID en Postgres sin distributed lock.';
      } else {
        responseText = `He procesado tu solicitud: "${text}". El AST del modelo se ha validado sin conflictos de herencia ni ciclos de cardinalidad.`;
      }

      setMessages(prev => [...prev, { sender: 'ai', text: responseText }]);
      setIsAnalyzing(false);

      if (onApplyRefactor) {
        onApplyRefactor(text);
      }
    }, 1000);
  };

  return (
    <div className="flex flex-col w-full min-h-[calc(100vh-4rem)] bg-[#0f131c] font-mono">
      {/* Header */}
      <div className="p-3 bg-[#181c24] border-b border-[#3c4a42] flex items-center justify-between">
        <div className="flex items-center gap-2">
          <span className="material-symbols-outlined text-[#d0bcff]">auto_awesome</span>
          <h2 className="font-heading text-sm text-white font-bold">AI Architect Console (Gemini &amp; ONNX AST)</h2>
        </div>
        <span className="text-[10px] px-2 py-0.5 bg-[#b090ff]/20 text-[#d0bcff] font-bold border border-[#d0bcff]/40">
          Model: AST-Copilot-v2
        </span>
      </div>

      <div className="flex-1 p-4 max-w-4xl w-full mx-auto flex flex-col justify-between space-y-4">
        {/* Messages feed */}
        <div className="space-y-3 overflow-y-auto max-h-[calc(100vh-16rem)]">
          {messages.map((m, idx) => (
            <div
              key={idx}
              className={`p-3 border text-xs leading-relaxed ${
                m.sender === 'ai'
                  ? 'bg-[#1c2028] border-[#3c4a42] text-[#dfe2ee]'
                  : 'bg-[#00422b]/30 border-[#4edea3] text-[#4edea3] ml-8'
              }`}
            >
              <div className="flex items-center gap-1.5 mb-1 text-[10px] text-[#86948a] uppercase font-bold">
                <span className="material-symbols-outlined text-xs">
                  {m.sender === 'ai' ? 'smart_toy' : 'person'}
                </span>
                <span>{m.sender === 'ai' ? 'AI Domain Architect' : 'You'}</span>
              </div>
              <p>{m.text}</p>
            </div>
          ))}

          {isAnalyzing && (
            <div className="p-3 bg-[#1c2028] border border-[#3c4a42] flex items-center gap-2 text-xs text-[#4edea3]">
              <span className="material-symbols-outlined text-sm animate-spin">refresh</span>
              <span>Sintetizando cambios en el AST y generando código JPA...</span>
            </div>
          )}
        </div>

        {/* Quick Prompts */}
        <div className="space-y-2">
          <span className="text-[10px] uppercase text-[#86948a] font-bold">Sugerencias Rápidas:</span>
          <div className="flex flex-wrap gap-1.5">
            {quickPrompts.map((p, idx) => (
              <button
                key={idx}
                onClick={() => handleSend(p)}
                className="px-2.5 py-1 bg-[#1c2028] hover:bg-[#262a33] border border-[#3c4a42] text-[11px] text-[#bbcabf] hover:text-[#4edea3] transition-colors text-left"
              >
                + {p}
              </button>
            ))}
          </div>
        </div>

        {/* Input form */}
        <form
          onSubmit={(e) => {
            e.preventDefault();
            handleSend();
          }}
          className="flex items-center gap-2 bg-[#181c24] p-2 border border-[#3c4a42]"
        >
          <input
            type="text"
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder="Pide una modificación de arquitectura (ej: Agregar cupones, normalizar tabla de pagos)..."
            className="flex-1 bg-[#0a0e16] border border-[#3c4a42] px-3 py-2 text-xs text-white focus:outline-none focus:border-[#4edea3]"
          />
          <button
            type="submit"
            className="px-4 py-2 bg-[#4edea3] text-black font-bold text-xs uppercase flex items-center gap-1.5 hover:brightness-110 active:scale-95 transition-all"
          >
            <span className="material-symbols-outlined text-xs">send</span>
            <span>Enviar</span>
          </button>
        </form>
      </div>
    </div>
  );
};
