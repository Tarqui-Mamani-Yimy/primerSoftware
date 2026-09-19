import { FormEvent, useId, useState } from 'react';
import { ArrowRight, Layers3, LockKeyhole, Mail, Mic, Sparkles } from 'lucide-react';
import { es } from '../../i18n/es';

interface LoginScreenProps {
  onContinue: (email: string, password: string) => Promise<void>;
}

/** Presentational local entry point until a canonical auth API exists. */
export function LoginScreen({ onContinue }: LoginScreenProps) {
  const emailId = useId();
  const passwordId = useId();
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');

  const [error, setError] = useState('');
  const [submitting, setSubmitting] = useState(false);
  const handleSubmit = async (event: FormEvent<HTMLFormElement>) => { event.preventDefault(); setError(''); setSubmitting(true); try { await onContinue(email, password); } catch (cause) { setError(cause instanceof Error ? cause.message : 'Unable to connect to the API.'); } finally { setSubmitting(false); } }; 

  return (
    <main aria-label={es.auth.enterWorkspace} className="min-h-screen bg-[#0a0e16] text-[#dfe2ee] px-5 py-8 sm:px-8 lg:px-12">
      <div className="mx-auto grid min-h-[calc(100vh-4rem)] max-w-6xl overflow-hidden rounded-3xl border border-[#303944] bg-[#101722] shadow-2xl shadow-black/30 lg:grid-cols-[1.12fr_.88fr]">
        <section className="relative overflow-hidden border-b border-[#303944] p-7 sm:p-10 lg:border-b-0 lg:border-r lg:p-12">
          <div className="pointer-events-none absolute -left-24 top-12 h-72 w-72 rounded-full bg-[#00c88c]/10 blur-3xl" />
          <div className="pointer-events-none absolute bottom-0 right-0 h-72 w-72 rounded-full bg-[#5b8cff]/10 blur-3xl" />
          <div className="relative flex h-full flex-col">
            <div className="flex items-center gap-3">
              <div className="grid h-11 w-11 place-items-center rounded-xl bg-gradient-to-br from-[#20d997] via-[#17b9dd] to-[#9f7aea] p-px">
                <div className="grid h-full w-full place-items-center rounded-[11px] bg-[#101722]"><Layers3 className="h-5 w-5 text-[#58e4ae]" aria-hidden="true" /></div>
              </div>
              <div>
                <p className="font-heading text-lg font-bold tracking-tight">AI UML Architect</p>
                <p className="text-xs text-[#98a6b8]">{es.auth.localWorkspace}</p>
              </div>
            </div>

            <div className="my-auto max-w-lg py-14 lg:py-0">
              <p className="mb-4 inline-flex items-center gap-2 rounded-full border border-[#2d6153] bg-[#12362c] px-3 py-1 text-xs font-semibold text-[#70e6b6]"><span className="h-1.5 w-1.5 rounded-full bg-[#70e6b6]" />{es.auth.workspaceReady}</p>
              <h1 className="font-heading text-4xl font-bold leading-tight tracking-tight text-white sm:text-5xl">{es.auth.headline}</h1>
              <p className="mt-5 max-w-md text-sm leading-7 text-[#aab6c5]">{es.auth.intro}</p>
              <div className="mt-9 grid gap-4 sm:grid-cols-3">
                {[
                  { icon: <Mic className="h-4 w-4" />, label: es.auth.voiceReady, detail: es.auth.localPipeline },
                  { icon: <Layers3 className="h-4 w-4" />, label: es.auth.structuredUml, detail: es.auth.canonicalModel },
                  { icon: <Sparkles className="h-4 w-4" />, label: es.auth.versionAware, detail: es.auth.safeIteration }
                ].map(item => (
                  <div key={item.label} className="rounded-xl border border-[#26313e] bg-[#141d29]/80 p-4">
                    <span className="mb-3 block text-[#61dcb1]" aria-hidden="true">{item.icon}</span>
                    <p className="text-xs font-bold text-[#e9eef7]">{item.label}</p>
                    <p className="mt-1 text-[11px] text-[#93a1b3]">{item.detail}</p>
                  </div>
                ))}
              </div>
            </div>

            <p className="text-xs text-[#6f7e91]">{es.auth.visibleProjects}</p>
          </div>
        </section>

        <section className="flex items-center p-7 sm:p-10 lg:p-12">
          <div className="mx-auto w-full max-w-sm">
            <p className="text-sm font-semibold text-[#62dcb1]">{es.auth.welcomeBack}</p>
            <h2 className="font-heading mt-2 text-3xl font-bold tracking-tight text-white">{es.auth.enterWorkspace}</h2>
            <p className="mt-3 text-sm leading-6 text-[#98a6b8]">{es.auth.useCredentials}</p>

            <form className="mt-8 space-y-5" onSubmit={handleSubmit}>
              <div>
                <label className="mb-2 block text-sm font-medium text-[#dfe7f1]" htmlFor={emailId}>{es.auth.email}</label>
                <div className="relative">
                  <Mail className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-[#8393a6]" aria-hidden="true" />
                  <input id={emailId} value={email} onChange={event => setEmail(event.target.value)} type="email" autoComplete="email" required className="w-full rounded-lg border border-[#364352] bg-[#0d141e] py-3 pl-10 pr-3 text-sm text-white outline-none transition focus:border-[#5ce0b2] focus:ring-2 focus:ring-[#5ce0b2]/20" />
                </div>
              </div>
              <div>
                <label className="mb-2 block text-sm font-medium text-[#dfe7f1]" htmlFor={passwordId}>{es.auth.password}</label>
                <div className="relative">
                  <LockKeyhole className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-[#8393a6]" aria-hidden="true" />
                  <input id={passwordId} value={password} onChange={event => setPassword(event.target.value)} type="password" autoComplete="current-password" placeholder={es.auth.passwordPlaceholder} className="w-full rounded-lg border border-[#364352] bg-[#0d141e] py-3 pl-10 pr-3 text-sm text-white outline-none transition placeholder:text-[#667588] focus:border-[#5ce0b2] focus:ring-2 focus:ring-[#5ce0b2]/20" />
                </div>
              </div>
              <button disabled={submitting} type="submit" className="group flex w-full items-center justify-center gap-2 rounded-lg bg-[#21c78d] px-4 py-3 text-sm font-bold text-[#062d20] transition hover:bg-[#52dfae] focus:outline-none focus:ring-2 focus:ring-[#8cf3cf] focus:ring-offset-2 focus:ring-offset-[#101722]">
                {submitting ? es.auth.signingIn : es.auth.signIn} <ArrowRight className="h-4 w-4 transition-transform group-hover:translate-x-0.5" aria-hidden="true" />
              </button>
            </form>
            {error && <p role="alert" className="mt-5 rounded-lg border border-red-700 bg-red-950/40 p-3 text-xs text-red-200">{error}</p>}
          </div>
        </section>
      </div>
    </main>
  );
}
