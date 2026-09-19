import { FormEvent, useId, useRef, useState } from 'react';
import { ArrowRight, ClipboardCheck, FolderKanban, LogOut, Plus, UsersRound } from 'lucide-react';
import { ApiError, AssignedProject, CreateProjectInput, CreatedProject } from '../../api/diagramApi';
import { es } from '../../i18n/es';
import { CreateProjectModal } from './CreateProjectModal';

interface ProjectDashboardProps {
  userName: string;
  projects: AssignedProject[];
  onOpenProject: (project: AssignedProject) => void;
  onSignOut: () => void;
  onCreateProject: (input: CreateProjectInput) => Promise<CreatedProject>;
  onJoinProject: (accessCode: string) => Promise<AssignedProject>;
}

function projectErrorMessage(cause: unknown, invalid: string, fallback: string): string {
  if (cause instanceof ApiError) {
    if (cause.status === 401) return es.projects.sessionExpired;
    if (cause.status === 404) return es.projects.joinCodeNotFound;
    if (cause.status === 400) return invalid;
  }
  return fallback;
}

export function ProjectDashboard({ userName, projects, onOpenProject, onSignOut, onCreateProject, onJoinProject }: ProjectDashboardProps) {
  const firstName = userName.split(' ')[0] || 'Architect';
  const joinCodeId = useId();
  const joinHelpId = useId();
  const joinErrorId = useId();
  const newProjectButtonRef = useRef<HTMLButtonElement>(null);
  const [isCreateOpen, setIsCreateOpen] = useState(false);
  const [isCreating, setIsCreating] = useState(false);
  const [createError, setCreateError] = useState('');
  const [createdProject, setCreatedProject] = useState<CreatedProject | undefined>();
  const [joinCode, setJoinCode] = useState('');
  const [joinError, setJoinError] = useState('');
  const [isJoining, setIsJoining] = useState(false);

  const openCreateModal = () => { setCreateError(''); setIsCreateOpen(true); };
  const closeCreateModal = () => { setIsCreateOpen(false); setCreateError(''); newProjectButtonRef.current?.focus(); };

  const handleCreateProject = async (input: CreateProjectInput) => {
    setCreateError('');
    setIsCreating(true);
    try {
      const created = await onCreateProject(input);
      setCreatedProject(created);
      closeCreateModal();
    } catch (cause) {
      setCreateError(projectErrorMessage(cause, es.projects.createInvalid, es.projects.createFailed));
    } finally {
      setIsCreating(false);
    }
  };

  const handleJoinProject = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    const code = joinCode.trim();
    if (!code) {
      setJoinError(es.projects.joinCodeRequired);
      return;
    }
    setJoinError('');
    setIsJoining(true);
    try {
      await onJoinProject(code.toUpperCase());
      setJoinCode('');
    } catch (cause) {
      setJoinError(projectErrorMessage(cause, es.projects.joinInvalid, es.projects.joinFailed));
    } finally {
      setIsJoining(false);
    }
  };

  return (
    <main className="min-h-screen bg-[#0a0e16] text-[#dfe2ee]">
      <header className="border-b border-[#273442] bg-[#0d141e]/95 backdrop-blur">
        <div className="mx-auto flex max-w-7xl items-center justify-between gap-4 px-5 py-4 sm:px-8">
          <div className="flex items-center gap-3">
            <div className="grid h-9 w-9 place-items-center rounded-lg bg-gradient-to-br from-[#20d997] via-[#17b9dd] to-[#9f7aea] p-px"><div className="grid h-full w-full place-items-center rounded-[7px] bg-[#101722]"><FolderKanban className="h-4 w-4 text-[#58e4ae]" aria-hidden="true" /></div></div>
            <span className="font-heading text-base font-bold tracking-tight text-white">AI UML Architect</span>
          </div>
          <button type="button" onClick={onSignOut} className="inline-flex items-center gap-2 rounded-lg border border-[#354352] px-3 py-2 text-xs font-semibold text-[#b9c5d3] transition hover:border-[#607185] hover:bg-[#172231] hover:text-white focus:outline-none focus:ring-2 focus:ring-[#70e6b6]">
            <LogOut className="h-3.5 w-3.5" aria-hidden="true" /> {es.projects.signOut}
          </button>
        </div>
      </header>

      <div className="mx-auto max-w-7xl px-5 py-10 sm:px-8 sm:py-14">
        <div className="flex flex-col justify-between gap-6 md:flex-row md:items-end">
          <div>
            <p className="text-sm font-semibold text-[#6ee6b6]">{es.projects.workspace}</p>
            <h1 className="font-heading mt-2 text-3xl font-bold tracking-tight text-white sm:text-4xl">{es.projects.greeting(firstName)}</h1>
            <p className="mt-3 max-w-xl text-sm leading-6 text-[#9aa9ba]">{es.projects.intro}</p>
          </div>
          <button ref={newProjectButtonRef} type="button" onClick={openCreateModal} aria-haspopup="dialog" aria-expanded={isCreateOpen} className="inline-flex items-center justify-center gap-2 rounded-lg bg-[#1fc88e] px-4 py-3 text-sm font-bold text-[#052d20] transition hover:bg-[#58e0b0] focus:outline-none focus:ring-2 focus:ring-[#8cf3cf] focus:ring-offset-2 focus:ring-offset-[#0a0e16]">
            <Plus className="h-4 w-4" aria-hidden="true" /> {es.projects.newProject}
          </button>
        </div>

        {createdProject && (
          <div role="status" className="mt-8 rounded-2xl border border-[#286853] bg-[#0f2a22] p-5 sm:p-6">
            <p className="flex items-center gap-2 text-sm font-semibold text-[#7defbd]">
              <ClipboardCheck className="h-4 w-4" aria-hidden="true" /> {es.projects.createSuccess(createdProject.name)}
            </p>
            <p className="mt-3 text-sm leading-6 text-[#a9c9bd]">{es.projects.shareCode}</p>
            <code className="mt-2 inline-block select-all rounded-lg border border-[#286853] bg-[#0a1f19] px-3 py-2 font-mono text-lg font-bold tracking-[0.25em] text-white">{createdProject.accessCode}</code>
          </div>
        )}

        <section className="mt-8 rounded-2xl border border-[#293746] bg-[#101923] p-5 sm:p-6" aria-labelledby="join-project-heading">
          <h2 id="join-project-heading" className="font-heading text-lg font-bold text-white">{es.projects.joinTitle}</h2>
          <p id={joinHelpId} className="mt-2 text-sm leading-6 text-[#9aa9ba]">{es.projects.joinHelp}</p>
          <form className="mt-4 flex flex-col gap-3 sm:flex-row sm:items-end" onSubmit={handleJoinProject} noValidate>
            <div className="flex-1">
              <label className="mb-2 block text-sm font-medium text-[#dfe7f1]" htmlFor={joinCodeId}>{es.projects.joinLabel}</label>
              <input
                id={joinCodeId}
                value={joinCode}
                onChange={event => { setJoinCode(event.target.value); if (joinError) setJoinError(''); }}
                maxLength={16}
                autoComplete="off"
                spellCheck={false}
                aria-invalid={joinError ? true : undefined}
                aria-describedby={joinError ? joinErrorId : joinHelpId}
                placeholder={es.projects.joinPlaceholder}
                className="w-full rounded-lg border border-[#364352] bg-[#0d141e] px-3 py-3 font-mono text-sm uppercase tracking-[0.2em] text-white outline-none transition placeholder:tracking-normal placeholder:text-[#667588] focus:border-[#5ce0b2] focus:ring-2 focus:ring-[#5ce0b2]/20"
              />
            </div>
            <button type="submit" disabled={isJoining} className="inline-flex items-center justify-center gap-2 rounded-lg border border-[#354352] px-4 py-3 text-sm font-semibold text-white transition hover:border-[#607185] hover:bg-[#172231] focus:outline-none focus:ring-2 focus:ring-[#70e6b6] disabled:cursor-not-allowed disabled:opacity-60">
              {isJoining ? es.projects.joinSubmitting : es.projects.joinSubmit}
            </button>
          </form>
          {joinError && <p id={joinErrorId} role="alert" className="mt-3 rounded-lg border border-red-700 bg-red-950/40 p-3 text-xs text-red-200">{joinError}</p>}
        </section>

        <section className="mt-10" aria-labelledby="assigned-projects-heading">
          <div className="mb-5 flex items-center justify-between gap-4">
            <h2 id="assigned-projects-heading" className="font-heading text-xl font-bold text-white">{es.projects.assigned} <span className="ml-1 text-sm font-normal text-[#8594a6]">({projects.length})</span></h2>
            <span className="rounded-full border border-[#2b4550] bg-[#102630] px-3 py-1 text-xs font-semibold text-[#6dd9ef]">{es.projects.apiData}</span>
          </div>
          {projects.length === 0 ? (
            <div className="rounded-2xl border border-dashed border-[#2b3b4c] bg-[#101923] p-10 text-center">
              <p className="text-sm font-semibold text-white">{es.projects.noProjects}</p>
              <p className="mx-auto mt-2 max-w-md text-sm leading-6 text-[#9aa9ba]">{es.projects.noProjectsHelp}</p>
            </div>
          ) : (
          <div className="grid gap-5 lg:grid-cols-3">
            {projects.map(project => (
              <article key={project.id} className="group flex min-h-80 flex-col overflow-hidden rounded-2xl border border-[#293746] bg-[#101923] transition hover:-translate-y-1 hover:border-[#4b6273] hover:shadow-xl hover:shadow-black/20">
                <div className="h-1.5 bg-gradient-to-r from-[#24d29b] to-[#26a6d7]" />
                <div className="flex flex-1 flex-col p-6">
                  <div className="flex items-start justify-between gap-3">
                    <span className="rounded-full border border-[#286853] bg-[#14392f] px-2.5 py-1 text-[11px] font-bold text-[#6ee6b6]">{project.role}</span>
                  </div>
                  <h3 className="font-heading mt-5 text-xl font-bold tracking-tight text-white">{project.name}</h3>
                  <p className="mt-2 text-sm leading-6 text-[#9aa9ba]">{project.description}</p>

                  <div className="mt-5 flex items-center justify-between gap-3">
                    <div className="flex items-center gap-2 text-xs text-[#98a7b8]"><UsersRound className="h-4 w-4 text-[#77a9d9]" aria-hidden="true" /><span>{es.projects.assignedMember}</span></div>
                    <span className="text-xs text-[#98a7b8]">{es.projects.diagrams(project.diagramCount)}</span>
                  </div>
                  <button type="button" onClick={() => onOpenProject(project)} className="mt-6 inline-flex w-full items-center justify-center gap-2 rounded-lg bg-[#1fc88e] px-4 py-3 text-sm font-bold text-[#052d20] transition hover:bg-[#58e0b0] focus:outline-none focus:ring-2 focus:ring-[#8cf3cf] focus:ring-offset-2 focus:ring-offset-[#101923]">
                    {es.projects.open} <ArrowRight className="h-4 w-4" aria-hidden="true" />
                  </button>
                </div>
              </article>
            ))}
          </div>
          )}
        </section>
      </div>

      <CreateProjectModal open={isCreateOpen} submitting={isCreating} error={createError} onClose={closeCreateModal} onSubmit={input => { void handleCreateProject(input); }} />
    </main>
  );
}
