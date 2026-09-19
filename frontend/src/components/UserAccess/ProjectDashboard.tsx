import { ArrowRight, FolderKanban, LogOut, Plus, UsersRound } from 'lucide-react';
import { AssignedProject } from '../../api/diagramApi';

interface ProjectDashboardProps {
  userName: string;
  projects: AssignedProject[];
  onOpenProject: (project: AssignedProject) => void;
  onSignOut: () => void;
}

export function ProjectDashboard({ userName, projects, onOpenProject, onSignOut }: ProjectDashboardProps) {
  const firstName = userName.split(' ')[0] || 'Architect';

  return (
    <main className="min-h-screen bg-[#0a0e16] text-[#dfe2ee]">
      <header className="border-b border-[#273442] bg-[#0d141e]/95 backdrop-blur">
        <div className="mx-auto flex max-w-7xl items-center justify-between gap-4 px-5 py-4 sm:px-8">
          <div className="flex items-center gap-3">
            <div className="grid h-9 w-9 place-items-center rounded-lg bg-gradient-to-br from-[#20d997] via-[#17b9dd] to-[#9f7aea] p-px"><div className="grid h-full w-full place-items-center rounded-[7px] bg-[#101722]"><FolderKanban className="h-4 w-4 text-[#58e4ae]" aria-hidden="true" /></div></div>
            <span className="font-heading text-base font-bold tracking-tight text-white">AI UML Architect</span>
          </div>
          <button type="button" onClick={onSignOut} className="inline-flex items-center gap-2 rounded-lg border border-[#354352] px-3 py-2 text-xs font-semibold text-[#b9c5d3] transition hover:border-[#607185] hover:bg-[#172231] hover:text-white focus:outline-none focus:ring-2 focus:ring-[#70e6b6]">
            <LogOut className="h-3.5 w-3.5" aria-hidden="true" /> Sign out
          </button>
        </div>
      </header>

      <div className="mx-auto max-w-7xl px-5 py-10 sm:px-8 sm:py-14">
        <div className="flex flex-col justify-between gap-6 md:flex-row md:items-end">
          <div>
            <p className="text-sm font-semibold text-[#6ee6b6]">Your workspace</p>
            <h1 className="font-heading mt-2 text-3xl font-bold tracking-tight text-white sm:text-4xl">Good to see you, {firstName}.</h1>
            <p className="mt-3 max-w-xl text-sm leading-6 text-[#9aa9ba]">Choose an assigned project and continue shaping its architecture with your team.</p>
          </div>
          <button type="button" disabled title="Project creation requires the future project API" className="inline-flex cursor-not-allowed items-center justify-center gap-2 rounded-lg border border-[#304151] bg-[#141f2c] px-4 py-3 text-sm font-semibold text-[#78889b]" aria-describedby="new-project-help">
            <Plus className="h-4 w-4" aria-hidden="true" /> New project
          </button>
        </div>
        <p id="new-project-help" className="mt-2 text-right text-xs text-[#6f7e91]">Project creation is not part of this first vertical slice.</p>

        <section className="mt-10" aria-labelledby="assigned-projects-heading">
          <div className="mb-5 flex items-center justify-between gap-4">
            <h2 id="assigned-projects-heading" className="font-heading text-xl font-bold text-white">Assigned projects <span className="ml-1 text-sm font-normal text-[#8594a6]">({projects.length})</span></h2>
            <span className="rounded-full border border-[#2b4550] bg-[#102630] px-3 py-1 text-xs font-semibold text-[#6dd9ef]">API data</span>
          </div>
          {projects.length === 0 ? (
            <div className="rounded-2xl border border-dashed border-[#2b3b4c] bg-[#101923] p-10 text-center">
              <p className="text-sm font-semibold text-white">No projects assigned yet</p>
              <p className="mx-auto mt-2 max-w-md text-sm leading-6 text-[#9aa9ba]">Your workspace is ready. Ask a teammate for an invite, then check back here. You can sign out anytime with the button above.</p>
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
                    <div className="flex items-center gap-2 text-xs text-[#98a7b8]"><UsersRound className="h-4 w-4 text-[#77a9d9]" aria-hidden="true" /><span>Assigned member</span></div>
                    <span className="text-xs text-[#98a7b8]">{project.diagramCount} diagrams</span>
                  </div>
                  <button type="button" onClick={() => onOpenProject(project)} className="mt-6 inline-flex w-full items-center justify-center gap-2 rounded-lg bg-[#1fc88e] px-4 py-3 text-sm font-bold text-[#052d20] transition hover:bg-[#58e0b0] focus:outline-none focus:ring-2 focus:ring-[#8cf3cf] focus:ring-offset-2 focus:ring-offset-[#101923]">
                    Open project <ArrowRight className="h-4 w-4" aria-hidden="true" />
                  </button>
                </div>
              </article>
            ))}
          </div>
          )}
        </section>
      </div>
    </main>
  );
}
